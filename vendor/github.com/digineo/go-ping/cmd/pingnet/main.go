package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	ping "github.com/digineo/go-ping"
	"gopkg.in/cheggaaa/pb.v1"
)

var (
	timeout  = 5 * time.Second
	attempts = 3
	poolSize = 2 * runtime.NumCPU()
	interval = 100 * time.Millisecond
	ifname   = ""
	bind6    = "::"
	bind4    = "0.0.0.0"
	size     = uint(56)
	force    bool
	verbose  bool
	pinger   *ping.Pinger
)

type workGenerator struct {
	ip  net.IP
	net *net.IPNet
}

func (w *workGenerator) size() uint64 {
	ones, bits := w.net.Mask.Size()
	return 1 << uint64(bits-ones)
}

func (w *workGenerator) each(callback func(net.IP) error) error {
	// adapted from http://play.golang.org/p/m8TNTtygK0
	inc := func(ip net.IP) net.IP {
		res := make(net.IP, len(ip))
		copy(res, ip)
		for j := len(res) - 1; j >= 0; j-- {
			res[j]++
			if res[j] > 0 {
				break
			}
		}
		return res
	}
	for ip := w.ip.Mask(w.net.Mask); w.net.Contains(ip); ip = inc(ip) {
		if err := callback(ip); err != nil {
			return err
		}
	}
	return nil
}

type result struct {
	addr net.IPAddr
	rtt  time.Duration
	err  error
}

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] CIDR [CIDR [...]]")
		flag.PrintDefaults()
	}

	flag.IntVar(&attempts, "c", attempts, "number of ping attempts per address")
	flag.DurationVar(&timeout, "w", timeout, "timeout for a single echo request")
	flag.DurationVar(&interval, "i", interval, "CIDR iteration interval")
	flag.UintVar(&size, "s", size, "size of additional payload data")
	flag.StringVar(&bind4, "4", bind4, "IPv4 bind address")
	flag.StringVar(&bind6, "6", bind6, "IPv6 bind address")
	flag.StringVar(&ifname, "I", ifname, "interface name/IPv6 zone")
	flag.IntVar(&poolSize, "P", poolSize, "concurrency level")
	flag.BoolVar(&force, "f", force, "sanity flag needed if you want to ping more than 4096 hosts (/20)")
	flag.BoolVar(&verbose, "v", verbose, "also print out unreachable addresses")
	flag.Parse()

	// simple error checking
	if bind4 == "" && bind6 == "" {
		log.Fatal("need at least an IPv4 (-bind4 flag) or IPv6 (-bind6 flag) address to bind to")
	}
	if poolSize <= 0 {
		log.Fatal("concurrency level (-P flag) must be > 0")
	}
	if attempts <= 0 {
		log.Fatal("number of ping attempts (-c flag) must be > 0")
	}

	// parse CIDR arguments
	total := uint64(0)
	generator := make([]*workGenerator, 0, flag.NArg())
	for _, cidr := range flag.Args() {
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Println(err)
			continue
		}
		w := &workGenerator{ip: ip, net: ipnet}
		generator = append(generator, w)
		total += w.size()
	}

	if total == 0 {
		// no (valid) CIDR argument given
		flag.Usage()
		os.Exit(1)
	} else if total > 4096 && !force {
		// expanding all arguments yields too many addresses
		log.Printf("You want to ping %d hosts. If that is correct, try again with -f flag", total)
		os.Exit(1)
	}

	if p, err := ping.New(bind4, bind6); err != nil {
		log.Fatal(err)
	} else {
		pinger = p
	}

	// prepare worker
	wg := &sync.WaitGroup{}
	wg.Add(poolSize)
	ips := make(chan net.IPAddr, poolSize)
	res := make(chan *result, poolSize)

	for i := 0; i < poolSize; i++ {
		go func() {
			for ip := range ips {
				var err error
				var rtt time.Duration
				for i := 1; ; i++ {
					rtt, err = pinger.PingAttempts(&ip, timeout, attempts)
					if err == nil || !strings.Contains(err.Error(), "no buffer space available") {
						break
					}
					time.Sleep(timeout * time.Duration(i))
				}

				res <- &result{addr: ip, rtt: rtt, err: err}
			}
			wg.Done()
		}()
	}

	// printer
	pr := &sync.WaitGroup{}
	pr.Add(1)
	go func() {
		bar := pb.New64(int64(total))
		bar.ShowBar = true
		bar.ShowTimeLeft = true
		bar.ShowCounters = true
		bar.Start()
		const clear = "\x1b[2K\r" // ansi delete line + CR

		for r := range res {
			bar.Increment()
			if r.err == nil {
				log.Printf("%s%s - rtt=%v", clear, r.addr.IP, r.rtt)
				bar.Update()
			} else if verbose {
				log.Printf("%s%s - %v", clear, r.addr, r.err)
				bar.Update()
			}
		}

		bar.Finish()
		pr.Done()
	}()

	// yield all IP addresses
	for _, g := range generator {
		g.each(func(ip net.IP) error {
			ips <- net.IPAddr{IP: ip, Zone: ifname}
			time.Sleep(interval)
			return nil
		})
	}

	// wait for worker and printer to finish
	close(ips)
	wg.Wait()
	close(res)
	pr.Wait()
}
