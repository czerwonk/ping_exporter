package main

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
)

var (
	ipv4Addr, ipv6Addr, ipv4AddrGoogle, ipv6AddrGoogle []net.IPAddr
)

func TestMain(m *testing.M) {
	// --- set up test addresses ---- //
	var err error
	ipv4Addr, err = net.DefaultResolver.LookupIPAddr(context.TODO(), "127.0.0.1")
	if err != nil || len(ipv4Addr) < 1 {
		log.Fatal("skipping test, cannot resolve 127.0.0.1 to net.IPAddr")
		return
	}

	ipv6Addr, err = net.DefaultResolver.LookupIPAddr(context.TODO(), "::1")
	if err != nil || len(ipv6Addr) < 1 {
		log.Fatal("skipping test, cannot resolve ::1 to net.IPAddr")
		return
	}

	ipv4AddrGoogle, err = net.DefaultResolver.LookupIPAddr(context.TODO(), "142.250.72.206")
	if err != nil || len(ipv4Addr) < 1 {
		log.Fatal("skipping test, cannot resolve 142.250.72.206 to net.IPAddr")
		return
	}

	ipv6AddrGoogle, err = net.DefaultResolver.LookupIPAddr(context.TODO(), "2607:f8b0:4005:810::200e")
	if err != nil || len(ipv6Addr) < 1 {
		log.Fatal("skipping test, cannot resolve 2607:f8b0:4005:810::200e to net.IPAddr")
		return
	}

	os.Exit(m.Run())
}

func Test_ipVersion_String(t *testing.T) {
	tests := []struct {
		name string
		ipv  ipVersion
		want string
	}{
		{
			"ipv6",
			ipv6,
			"6",
		},
		{
			"ipv4",
			ipv4,
			"4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ipv.String(); got != tt.want {
				t.Errorf("IPVersion.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getIPVersion(t *testing.T) {
	tests := []struct {
		name string
		addr net.IPAddr
		want ipVersion
	}{
		{
			"ipv4",
			ipv4Addr[0],
			ipv4,
		},
		{
			"ipv6",
			ipv6Addr[0],
			ipv6,
		},
		{
			"ipv4-google",
			ipv4AddrGoogle[0],
			ipv4,
		},
		{
			"ipv6-google",
			ipv6AddrGoogle[0],
			ipv6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIPVersion(tt.addr); got != tt.want {
				t.Errorf("getIPVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_target_nameForIP(t *testing.T) {
	tests := []struct {
		name string
		addr net.IPAddr
		want string
	}{
		{
			"ipv4-localhost",
			ipv4Addr[0],
			"testhost.com 127.0.0.1 4",
		},
		{
			"ipv6-localhost",
			ipv6Addr[0],
			"testhost.com ::1 6",
		},
		{
			"ipv4-google",
			ipv4AddrGoogle[0],
			"testhost.com 142.250.72.206 4",
		},
		{
			"ipv4-google",
			ipv6AddrGoogle[0],
			"testhost.com 2607:f8b0:4005:810::200e 6",
		},
	}
	for _, tt := range tests {
		tr := &target{
			host:      "testhost.com",
			addresses: []net.IPAddr{},
			delay:     0,
			resolver:  &net.Resolver{},
			mutex:     sync.Mutex{},
		}
		t.Run(tt.name, func(t *testing.T) {
			if got := tr.nameForIP(tt.addr); got != tt.want {
				t.Errorf("target.nameForIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
