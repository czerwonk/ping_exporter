// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/czerwonk/ping_exporter/config"
	mon "github.com/digineo/go-ping/monitor"
	log "github.com/sirupsen/logrus"
)

// ipVersion represents the IP protocol version of an address
type ipVersion uint8

type target struct {
	host      string
	addresses []net.IPAddr
	delay     time.Duration
	resolver  Resolver
	mutex     sync.Mutex
}

type targets struct {
	t     []*target
	mutex sync.RWMutex
}

func (t *targets) SetTargets(tar []*target) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.t = tar
}

func (t *targets) Contains(tar *target) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	for _, ta := range t.t {
		if ta.host == tar.host {
			return true
		}
	}
	return false
}

func (t *targets) Get(host string) *target {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	for _, ta := range t.t {
		if ta.host == host {
			return ta
		}
	}
	return nil
}

func (t *targets) Targets() []*target {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	ret := make([]*target, len(t.t))
	copy(ret, t.t)
	return ret
}

type targetOpts struct {
	disableIPv4 bool
	disableIPv6 bool
}

const (
	ipv4 ipVersion = 4
	ipv6 ipVersion = 6
)

func (t *target) removeFromMonitor(monitor *mon.Monitor) {
	for _, addr := range t.addresses {
		monitor.RemoveTarget(t.nameForIP(addr))
	}
}

func (t *target) addOrUpdateMonitor(monitor *mon.Monitor, opts targetOpts, cfg *config.Config) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	ctx := context.Background()
	if cfg.DNS.Timeout.Duration() != time.Duration(0*time.Second) {
		log.Infof("DNS timeout enabled: using %+v", cfg.DNS.Timeout)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), cfg.DNS.Timeout.Duration())
		defer cancel()
	}
	addrs, err := t.resolver.LookupIPAddr(ctx, t.host)
	if err != nil {
		return fmt.Errorf("error resolving target '%s': %w", t.host, err)
	}

	var sanitizedAddrs []net.IPAddr
	for _, addr := range addrs {
		if getIPVersion(addr) == ipv6 && opts.disableIPv6 {
			log.Infof("IPv6 disabled: skipping target for host %s (%v)", t.host, addr)
			continue
		}
		if getIPVersion(addr) == ipv4 && opts.disableIPv4 {
			log.Infof("IPv4 disabled: skipping target for host %s (%v)", t.host, addr)
			continue
		}
		sanitizedAddrs = append(sanitizedAddrs, addr)
	}

	for _, addr := range sanitizedAddrs {
		err := t.addIfNew(addr, monitor)
		if err != nil {
			return err
		}
	}

	t.cleanUp(sanitizedAddrs, monitor)
	t.addresses = sanitizedAddrs

	return nil
}

func (t *target) addIfNew(addr net.IPAddr, monitor *mon.Monitor) error {
	if isIPAddrInSlice(addr, t.addresses) {
		return nil
	}

	return t.add(addr, monitor)
}

func (t *target) cleanUp(addr []net.IPAddr, monitor *mon.Monitor) {
	for _, o := range t.addresses {
		if !isIPAddrInSlice(o, addr) {
			name := t.nameForIP(o)
			log.Infof("removing target for host %s (%v)", t.host, o)
			monitor.RemoveTarget(name)
		}
	}
}

func (t *target) add(addr net.IPAddr, monitor *mon.Monitor) error {
	name := t.nameForIP(addr)
	log.Infof("adding target for host %s (%v)", t.host, addr)

	return monitor.AddTargetDelayed(name, addr, t.delay)
}

func (t *target) nameForIP(addr net.IPAddr) string {
	return fmt.Sprintf("%s %s %s", t.host, addr.IP, getIPVersion(addr))
}

func isIPAddrInSlice(ipa net.IPAddr, slice []net.IPAddr) bool {
	for _, x := range slice {
		if x.IP.Equal(ipa.IP) {
			return true
		}
	}

	return false
}

// getIPVersion returns the version of IP protocol used for a given address
func getIPVersion(addr net.IPAddr) ipVersion {
	if addr.IP.To4() == nil {
		return ipv6
	}

	return ipv4
}

// String converts ipVersion to a string represention of the IP version used (i.e. "4" or "6")
func (ipv ipVersion) String() string {
	return strconv.Itoa(int(ipv))
}
