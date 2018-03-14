package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	ping "github.com/digineo/go-ping"
)

var opts = struct {
	timeout         time.Duration
	interval        time.Duration
	payloadSize     uint
	statBufferSize  uint
	bind4           string
	bind6           string
	dests           []*destination
	resolverTimeout time.Duration
}{
	timeout:         1000 * time.Millisecond,
	interval:        1000 * time.Millisecond,
	bind4:           "0.0.0.0",
	bind6:           "::",
	payloadSize:     56,
	statBufferSize:  50,
	resolverTimeout: 1500 * time.Millisecond,
}

var (
	pinger *ping.Pinger
	tui    *userInterface
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] host [host [...]]")
		flag.PrintDefaults()
	}

	flag.DurationVar(&opts.timeout, "timeout", opts.timeout, "timeout for a single echo request")
	flag.DurationVar(&opts.interval, "interval", opts.interval, "polling interval")
	flag.UintVar(&opts.payloadSize, "s", opts.payloadSize, "size of payload in bytes")
	flag.UintVar(&opts.statBufferSize, "buf", opts.statBufferSize, "buffer size for statistics")
	flag.StringVar(&opts.bind4, "bind4", opts.bind4, "IPv4 bind address")
	flag.StringVar(&opts.bind6, "bind6", opts.bind6, "IPv6 bind address")
	flag.DurationVar(&opts.resolverTimeout, "resolve", opts.resolverTimeout, "timeout for DNS lookups")
	flag.Parse()

	log.SetFlags(0)

	for _, host := range flag.Args() {
		remotes, err := resolve(host, opts.resolverTimeout)
		if err != nil {
			log.Printf("error resolving host %s: %v", host, err)
			continue
		}

		for _, remote := range remotes {
			if v4 := remote.IP.To4() != nil; v4 && opts.bind4 == "" || !v4 && opts.bind6 == "" {
				continue
			}

			ipaddr := remote // need to create a copy
			dst := destination{
				host:   host,
				remote: &ipaddr,
				history: &history{
					results: make([]time.Duration, opts.statBufferSize),
				},
			}

			opts.dests = append(opts.dests, &dst)
		}
	}

	if instance, err := ping.New(opts.bind4, opts.bind6); err == nil {
		if instance.PayloadSize() != uint16(opts.payloadSize) {
			instance.SetPayloadSize(uint16(opts.payloadSize))
		}
		pinger = instance
		defer pinger.Close()
	} else {
		panic(err)
	}

	go work()

	tui = buildTUI(opts.dests)
	go tui.update(time.Second)

	if err := tui.Run(); err != nil {
		panic(err)
	}
}

func work() {
	for {
		for i, u := range opts.dests {
			go func(u *destination, i int) {
				u.ping(pinger)
			}(u, i)
		}
		time.Sleep(opts.interval)
	}
}
