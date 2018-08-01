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
	dns       string
	addresses []net.IP
	delay     time.Duration
	mutex     sync.Mutex
}

func (t *target) addOrUpdateMonitor(monitor *mon.Monitor) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

        if (len(t.dns) > 0) {
             dialer := func (ctx context.Context, network, address string) (net.Conn,error) {
                 d := net.Dialer{}
                 return d.DialContext(ctx, "udp", t.dns)
             }
	     net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: dialer}
        }

	addrs, err := net.LookupIP(t.host)
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

func (t *target) addIfNew(addr net.IP, monitor *mon.Monitor) error {
	if isIPInSlice(addr, t.addresses) {
		return nil
	}

	return t.add(addr, monitor)
}

func (t *target) cleanUp(new []net.IP, monitor *mon.Monitor) {
	for _, o := range t.addresses {
		if !isIPInSlice(o, new) {
			name := t.nameForIP(o)
			log.Infof("removing target for host %s (%v)", t.host, o)
			monitor.RemoveTarget(name)
		}
	}
}

func (t *target) add(addr net.IP, monitor *mon.Monitor) error {
	name := t.nameForIP(addr)
	log.Infof("adding target for host %s (%v)", t.host, addr)
	return monitor.AddTargetDelayed(name, net.IPAddr{IP: addr, Zone: ""}, t.delay)
}

func (t *target) nameForIP(addr net.IP) string {
	name := fmt.Sprintf("%s %s ", t.host, addr)

	if addr.To4() == nil {
		name += "6"
	} else {
		name += "4"
	}

	return name
}

func isIPInSlice(ip net.IP, slice []net.IP) bool {
	for _, x := range slice {
		if x.Equal(ip) {
			return true
		}
	}

	return false
}
