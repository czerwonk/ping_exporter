package monitor

import (
	"net"
	"sync"
	"time"

	ping "github.com/digineo/go-ping"
)

// Target is a unit of work
type Target struct {
	pinger   *ping.Pinger
	addr     net.IPAddr
	interval time.Duration
	timeout  time.Duration
	stop     chan struct{}
	history  History
	wg       sync.WaitGroup
}

// newTarget starts a new monitoring goroutine
func newTarget(interval, timeout, startupDelay time.Duration, pinger *ping.Pinger, addr net.IPAddr) (*Target, error) {
	n := &Target{
		pinger:   pinger,
		addr:     addr,
		interval: interval,
		timeout:  timeout,
		stop:     make(chan struct{}),
	}
	n.wg.Add(1)
	go n.run(startupDelay)
	return n, nil
}

func (n *Target) run(startupDelay time.Duration) {
	if startupDelay > 0 {
		select {
		case <-time.After(startupDelay):
		case <-n.stop:
		}
	}

	tick := time.NewTicker(n.interval)
	for {
		select {
		case <-n.stop:
			tick.Stop()
			n.wg.Done()
			return
		case <-tick.C:
			go n.ping()
		}
	}
}

// Stop gracefully stops the monitoring.
func (n *Target) Stop() {
	close(n.stop)
	n.wg.Wait()
}

// ComputeAndClear returns the computed ping metrics for this node and clears the result set.
func (n *Target) ComputeAndClear() *Metrics {
	return n.history.ComputeAndClear()
}

func (n *Target) ping() {
	n.history.AddResult(n.pinger.Ping(&n.addr, n.timeout))
}
