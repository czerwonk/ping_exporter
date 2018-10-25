package monitor

import (
	"net"
	"sync"
	"time"

	"github.com/digineo/go-ping"
)

// Monitor manages the goroutines responsible for collecting Ping RTT data.
type Monitor struct {
	HistorySize int // Number of results per target to keep

	pinger   *ping.Pinger
	interval time.Duration
	targets  map[string]*Target
	mtx      sync.RWMutex
	timeout  time.Duration
}

const defaultHistorySize = 10

// New creates and configures a new Ping instance. You need to call
// AddTarget()/RemoveTarget() to manage monitored targets.
func New(pinger *ping.Pinger, interval, timeout time.Duration) *Monitor {
	return &Monitor{
		pinger:      pinger,
		interval:    interval,
		timeout:     timeout,
		targets:     make(map[string]*Target),
		HistorySize: defaultHistorySize,
	}
}

// Stop brings the monitoring gracefully to a halt.
func (p *Monitor) Stop() {
	p.mtx.Lock()
	for id := range p.targets {
		p.removeTarget(id)
	}
	p.pinger.Close()
	p.mtx.Unlock()
}

// AddTarget adds a target to the monitored list. If the target with the given
// ID already exists, it is removed first and then readded. This allows
// the easy restart of the monitoring.
func (p *Monitor) AddTarget(key string, addr net.IPAddr) (err error) {
	return p.AddTargetDelayed(key, addr, 0)
}

// AddTargetDelayed is AddTarget with a startup delay
func (p *Monitor) AddTargetDelayed(key string, addr net.IPAddr, startupDelay time.Duration) (err error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	target, err := newTarget(p.interval, p.timeout, startupDelay, p.HistorySize, p.pinger, addr)
	if err != nil {
		return err
	}
	p.removeTarget(key)
	p.targets[key] = target
	return
}

// RemoveTarget removes a target from the monitoring list.
func (p *Monitor) RemoveTarget(key string) {
	p.mtx.Lock()
	p.removeTarget(key)
	p.mtx.Unlock()
}

// Stops monitoring a target and removes it from the list (if the list includes
// the target). Needs to be locked externally!
func (p *Monitor) removeTarget(key string) {
	target, found := p.targets[key]
	if !found {
		return
	}
	target.Stop()
	delete(p.targets, key)
}

// ExportAndClear calculates the metrics for each monitored target, cleans the result set and
// returns it as a simple map.
func (p *Monitor) ExportAndClear() map[string]*Metrics {
	return p.export(true)
}

// Export calculates the metrics for each monitored target and returns it as a simple map.
func (p *Monitor) Export() map[string]*Metrics {
	return p.export(false)
}

func (p *Monitor) export(clear bool) map[string]*Metrics {
	m := make(map[string]*Metrics)

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	for id, target := range p.targets {
		if metrics := target.Compute(clear); metrics != nil {
			m[id] = metrics
		}
	}
	return m
}
