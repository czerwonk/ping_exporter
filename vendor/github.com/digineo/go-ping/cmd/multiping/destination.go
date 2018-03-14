package main

import (
	"log"
	"math"
	"net"
	"sync"
	"time"

	ping "github.com/digineo/go-ping"
)

type history struct {
	received int
	lost     int
	results  []time.Duration // ring, start index = .received%len
	mtx      sync.RWMutex
}

type destination struct {
	host    string
	remote  *net.IPAddr
	display string
	*history
}

type stat struct {
	pktSent int
	pktLoss float64
	last    time.Duration
	best    time.Duration
	worst   time.Duration
	mean    time.Duration
	stddev  time.Duration
}

func (u *destination) ping(pinger *ping.Pinger) {
	rtt, err := pinger.Ping(u.remote, opts.timeout)
	if err != nil {
		log.Printf("[yellow]%s[white]: %v", u.host, err)
	}
	u.addResult(rtt, err)
}

func (s *history) addResult(rtt time.Duration, err error) {
	s.mtx.Lock()
	if err == nil {
		s.results[s.received%len(s.results)] = rtt
		s.received++
	} else {
		s.lost++
	}
	s.mtx.Unlock()
}

func (s *history) compute() (st stat) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if s.received == 0 {
		if s.lost > 0 {
			st.pktLoss = 1.0
		}
		return
	}

	collection := s.results[:]
	st.pktSent = s.received + s.lost
	size := len(s.results)
	st.last = collection[(s.received-1)%size]

	// we don't yet have filled the buffer
	if s.received <= size {
		collection = s.results[:s.received]
		size = s.received
	}

	st.pktLoss = float64(s.lost) / float64(s.received+s.lost)
	st.best, st.worst = collection[0], collection[0]

	total := time.Duration(0)
	for _, rtt := range collection {
		if rtt < st.best {
			st.best = rtt
		}
		if rtt > st.worst {
			st.worst = rtt
		}
		total += rtt
	}

	st.mean = time.Duration(float64(total) / float64(size))

	stddevNum := float64(0)
	for _, rtt := range collection {
		stddevNum += math.Pow(float64(rtt-st.mean), 2)
	}
	st.stddev = time.Duration(math.Sqrt(stddevNum / float64(size)))

	return
}
