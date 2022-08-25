package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	mon "github.com/digineo/go-ping/monitor"
	log "github.com/sirupsen/logrus"
)

// ipVersion represents the IP protocol version of an address
type ipVersion uint8

type target struct {
	addressScopeName string
	host             string
	addresses        []net.IPAddr
	delay            time.Duration
	resolver         *net.Resolver
	mutex            sync.Mutex
}

const (
	ipv4 ipVersion = 4
	ipv6 ipVersion = 6
)

func (t *target) addOrUpdateMonitor(monitor *mon.Monitor, disableIPv6 bool) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	addrs, err := t.resolver.LookupIPAddr(context.Background(), t.host)
	if err != nil {
		return fmt.Errorf("error resolving target: %w", err)
	}

	var sanitizedAddrs []net.IPAddr
	if disableIPv6 {
		for _, addr := range addrs {
			if getIPVersion(addr) == ipv6 {
				log.Infof("IPv6 disabled: skipping target for host %s (%v)", t.host, addr)
				continue
			}
			sanitizedAddrs = append(sanitizedAddrs, addr)
		}
	} else {
		sanitizedAddrs = addrs
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
	log.Infof("adding target for host %s (%v)", t.host+" "+t.addressScopeName, addr)

	return monitor.AddTargetDelayed(name+" "+t.addressScopeName, addr, t.delay)
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
