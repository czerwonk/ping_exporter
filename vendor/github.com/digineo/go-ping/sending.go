package ping

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// PingAttempts sends ICMP echo requests with a timeout per request, retrying upto `attempt` times .
// Will finish early on success and return the round trip time of the last ping.
func (pinger *Pinger) PingAttempts(remote *net.IPAddr, timeout time.Duration, attempts int) (rtt time.Duration, err error) {
	if attempts < 1 {
		err = errors.New("zero attempts")
	} else {
		for i := 0; i < attempts; i++ {
			if rtt, err = pinger.Ping(remote, timeout); err == nil {
				break // success
			}
		}
	}
	return
}

// Ping sends a single Echo Request and waits for an answer. It returns
// the round trip time (RTT) if a reply is received in time.
func (pinger *Pinger) Ping(remote *net.IPAddr, timeout time.Duration) (time.Duration, error) {
	if timeout <= 0 {
		return 0, errors.New("zero timeout")
	}

	seq := uint16(atomic.AddUint32(&sequence, 1))
	req := request{
		wait: make(chan struct{}),
	}

	pinger.payloadMu.RLock()
	defer pinger.payloadMu.RUnlock()

	// build packet
	wm := icmp.Message{
		Code: 0,
		Body: &icmp.Echo{
			ID:   int(pinger.id),
			Seq:  int(seq),
			Data: pinger.payload,
		},
	}

	// Protocol specifics
	var conn *icmp.PacketConn
	var lock *sync.Mutex
	if remote.IP.To4() != nil {
		wm.Type = ipv4.ICMPTypeEcho
		conn = pinger.conn4
		lock = &pinger.write4
	} else {
		wm.Type = ipv6.ICMPTypeEchoRequest
		conn = pinger.conn6
		lock = &pinger.write6
	}

	// serialize packet
	wb, err := wm.Marshal(nil)
	if err != nil {
		return 0, err
	}

	// enqueue in currently running requests
	pinger.mtx.Lock()
	pinger.requests[seq] = &req
	pinger.mtx.Unlock()

	// start measurement (tStop is set in the receiving end)
	lock.Lock()
	req.tStart = time.Now()

	// send request
	if _, e := conn.WriteTo(wb, remote); e != nil {
		req.respond(e, nil)
	}
	lock.Unlock()

	// wait for answer
	select {
	case <-req.wait:
		err = req.result
	case <-time.After(timeout):
		err = &timeoutError{}
	}

	// dequeue request
	pinger.mtx.Lock()
	delete(pinger.requests, seq)
	pinger.mtx.Unlock()

	if err != nil {
		return 0, err
	}
	return req.roundTripTime()
}
