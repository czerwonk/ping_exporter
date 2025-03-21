package main

import (
	"context"
	"net"
)

type Resolver interface {
	// LookupIP resolves a host to its IP addresses.
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}
