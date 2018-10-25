package monitor

import (
	"math"
	"sync"
	"time"
)

// Result stores the information about a single ping, in particular
// the round-trip time or whether the packet was lost.
type Result struct {
	RTT  time.Duration
	Lost bool
}

// History represents the ping history for a single node/device.
type History struct {
	results  []Result
	count    int
	position int
	sync.RWMutex
}

// NewHistory creates a new History object with a specific capacity
func NewHistory(capacity int) History {
	return History{
		results: make([]Result, capacity),
	}
}

// AddResult saves a ping result into the internal history.
func (h *History) AddResult(rtt time.Duration, err error) {
	h.Lock()

	h.results[h.position] = Result{RTT: rtt, Lost: err != nil}
	h.position = (h.position + 1) % cap(h.results)

	if h.count < cap(h.results) {
		h.count++
	}

	h.Unlock()
}

func (h *History) clear() {
	h.count = 0
	h.position = 0
}

// ComputeAndClear aggregates the result history into a single data point and clears the result set.
func (h *History) ComputeAndClear() *Metrics {
	h.Lock()
	result := h.compute()
	h.clear()
	h.Unlock()
	return result
}

// Compute aggregates the result history into a single data point.
func (h *History) Compute() *Metrics {
	h.RLock()
	defer h.RUnlock()
	return h.compute()
}

func (h *History) compute() *Metrics {
	numFailure := 0
	numTotal := h.count
	µsPerMs := 1.0 / float64(time.Millisecond)

	if numTotal == 0 {
		return nil
	}

	data := make([]float64, 0, numTotal)
	var best, worst, mean, stddev, total, sumSquares float64
	var extremeFound bool

	for i := 0; i < numTotal; i++ {
		curr := &h.results[i]
		if curr.Lost {
			numFailure++
		} else {
			rtt := float64(curr.RTT) * µsPerMs
			data = append(data, rtt)

			if !extremeFound || rtt < best {
				best = rtt
			}
			if !extremeFound || rtt > worst {
				worst = rtt
			}

			extremeFound = true
			total += rtt
		}
	}

	size := float64(numTotal - numFailure)
	mean = total / size
	for _, rtt := range data {
		sumSquares += math.Pow(rtt-mean, 2)
	}
	stddev = math.Sqrt(sumSquares / size)

	return &Metrics{
		PacketsSent: numTotal,
		PacketsLost: numFailure,
		Best:        float32(best),
		Worst:       float32(worst),
		Mean:        float32(mean),
		StdDev:      float32(stddev),
	}
}
