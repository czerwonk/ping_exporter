package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	mon "github.com/digineo/go-ping/monitor"
	"github.com/prometheus/common/log"
)

type target struct {
	host      string
	addresses []net.IPAddr
	delay     time.Duration
	resolver  *net.Resolver
	mutex     sync.Mutex
}

func (t *target) addOrUpdateMonitor(monitor *mon.Monitor) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	addrs, err := t.resolver.LookupIPAddr(context.Background(), t.host)
	if err != nil {
		return fmt.Errorf("error resolving target: %v", err)
	}

	for _, addr := range addrs {
		err := t.addIfNew(addr, monitor)
		if err != nil {
			return err
		}
	}

	t.cleanUp(addrs, monitor)
	t.addresses = addrs

	return nil
}

func (t *target) addIfNew(addr net.IPAddr, monitor *mon.Monitor) error {
	if isIPAddrInSlice(addr, t.addresses) {
		return nil
	}
	return t.add(addr, monitor)
}

func (t *target) cleanUp(new []net.IPAddr, monitor *mon.Monitor) {
	for _, o := range t.addresses {
		if !isIPAddrInSlice(o, new) {
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
	v := 4
	if addr.IP.To4() == nil {
		v = 6
	}
	return fmt.Sprintf("%s %s %d", t.host, addr.IP, v)
}

func isIPAddrInSlice(ipa net.IPAddr, slice []net.IPAddr) bool {
	for _, x := range slice {
		if x.IP.Equal(ipa.IP) {
			return true
		}
	}
	return false
}
