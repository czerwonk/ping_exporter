package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	ping "github.com/digineo/go-ping"
)

var (
	args           []string
	attempts       uint = 3
	timeout             = time.Second
	proto4, proto6 bool
	size           uint = 56
	bind           string

	destination string
	remoteAddr  *net.IPAddr
	pinger      *ping.Pinger
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] host [host [...]]")
		flag.PrintDefaults()
	}

	flag.UintVar(&attempts, "attempts", attempts, "number of attempts")
	flag.DurationVar(&timeout, "timeout", timeout, "timeout for a single echo request")
	flag.UintVar(&size, "s", size, "size of additional payload data")
	flag.BoolVar(&proto4, "4", proto4, "use IPv4 (mutually exclusive with -6)")
	flag.BoolVar(&proto6, "6", proto6, "use IPv6 (mutually exclusive with -4)")
	flag.StringVar(&bind, "bind", "", "IPv4 or IPv6 bind address (defaults to 0.0.0.0 for IPv4 and :: for IPv6)")
	flag.Parse()

	if proto4 == proto6 {
		log.Fatalf("need exactly one of -4 and -6 flags")
	}

	if bind == "" {
		if proto4 {
			bind = "0.0.0.0"
		} else if proto6 {
			bind = "::"
		}
	}

	args := flag.Args()
	destination := args[0]

	if proto4 {
		if r, err := net.ResolveIPAddr("ip4", destination); err != nil {
			panic(err)
		} else {
			remoteAddr = r
		}

		if p, err := ping.New(bind, ""); err != nil {
			panic(err)
		} else {
			pinger = p
		}
	} else if proto6 {
		if r, err := net.ResolveIPAddr("ip6", destination); err != nil {
			panic(err)
		} else {
			remoteAddr = r
		}

		if p, err := ping.New("", bind); err != nil {
			panic(err)
		} else {
			pinger = p
		}
	}
	defer pinger.Close()

	if pinger.PayloadSize() != uint16(size) {
		pinger.SetPayloadSize(uint16(size))
	}

	if remoteAddr.IP.IsLinkLocalMulticast() {
		multicastPing()
	} else {
		unicastPing()
	}
}

func unicastPing() {
	rtt, err := pinger.PingAttempts(remoteAddr, timeout, int(attempts))

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("ping %s (%s) rtt=%v\n", destination, remoteAddr, rtt)
}

func multicastPing() {
	fmt.Printf("multicast ping to %s (%s)\n", args[0], destination)

	responses, err := pinger.PingMulticast(remoteAddr, timeout)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for response := range responses {
		fmt.Printf("%+v\n", response)
	}
}
