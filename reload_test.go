// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"net"
	"testing"
	"time"

	mon "github.com/digineo/go-ping/monitor"

	"github.com/czerwonk/ping_exporter/config"
)

type staticResolver struct {
	addrs []net.IPAddr
}

func (r staticResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r.addrs, nil
}

func TestRefreshDNSUsesReloadedConfig(t *testing.T) {
	t.Cleanup(func() { activeConfig.Store(nil) })

	resolver := staticResolver{addrs: []net.IPAddr{ipv4Addr[0], ipv6Addr[0]}}
	// The pinger is never used: the ping interval is longer than the test.
	monitor := mon.New(nil, time.Hour, time.Second)
	tgts := &targets{}

	initial := &config.Config{Targets: []config.TargetConfig{{Addr: "example.test"}}}
	if err := upsertTargets(tgts, resolver, initial, monitor); err != nil {
		t.Fatal(err)
	}
	activeConfig.Store(initial)
	collector := NewPingCollector(false, rttInSeconds, monitor, initial)

	reloaded := &config.Config{Targets: []config.TargetConfig{{Addr: "example.test"}}}
	reloaded.Options.DisableIPv6 = true
	if err := applyConfig(tgts, resolver, reloaded, monitor, collector); err != nil {
		t.Fatal(err)
	}
	assertAddresses(t, tgts, "after reload", ipv4Addr[0])

	refreshDNS(tgts, monitor)
	assertAddresses(t, tgts, "after DNS refresh", ipv4Addr[0])
}

func assertAddresses(t *testing.T, tgts *targets, when string, want ...net.IPAddr) {
	t.Helper()
	all := tgts.Targets()
	if len(all) != 1 {
		t.Fatalf("%s: got %d targets, want 1", when, len(all))
	}
	got := all[0].addresses
	if len(got) != len(want) {
		t.Fatalf("%s: got addresses %v, want %v", when, got, want)
	}
	for i := range want {
		if !got[i].IP.Equal(want[i].IP) {
			t.Fatalf("%s: got addresses %v, want %v", when, got, want)
		}
	}
}
