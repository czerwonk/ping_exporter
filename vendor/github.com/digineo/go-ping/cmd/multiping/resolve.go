package main

import (
	"context"
	"net"
	"strings"
	"time"
)

func resolve(addr string, timeout time.Duration) ([]net.IPAddr, error) {
	if strings.ContainsRune(addr, '%') {
		ipaddr, err := net.ResolveIPAddr("ip", addr)
		if err != nil {
			return nil, err
		}
		return []net.IPAddr{*ipaddr}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return net.DefaultResolver.LookupIPAddr(ctx, addr)
}
