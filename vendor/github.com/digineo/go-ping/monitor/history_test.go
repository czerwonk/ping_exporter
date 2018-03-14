package monitor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkAddResult(b *testing.B) {
	h := &History{}
	for i := 0; i < b.N; i++ {
		h.AddResult(time.Duration(i), nil) // 1 allocc
	}
}

func BenchmarkCompute(b *testing.B) {
	h := &History{}
	for i := 0; i < b.N; i++ {
		h.AddResult(time.Duration(i), nil) // 1 alloc
		h.Compute()                        // 2 allocs
	}
}

func TestCompute(t *testing.T) {
	assert := assert.New(t)
	const dur = 100 * time.Millisecond
	err := fmt.Errorf("i/o timeout")

	{ // empty list
		h := &History{}
		assert.Nil(h.Compute())
	}

	{ // populate with 5 entries
		h := &History{}
		h.AddResult(0, nil)
		h.AddResult(dur, nil)
		h.AddResult(dur, nil)
		h.AddResult(0, err)
		h.AddResult(dur, nil)

		assert.Len(h.results, 5)
		assert.EqualValues(1, h.Compute().PacketsLost)
	}

	{
		// test zero variance
		h := &History{}
		h.AddResult(dur, nil)
		h.AddResult(dur, nil)
		h.AddResult(0, err)

		metrics := h.Compute()
		assert.EqualValues(100, metrics.Best)
		assert.EqualValues(100, metrics.Worst)
		assert.EqualValues(100, metrics.Mean)
		assert.EqualValues(0, metrics.StdDev)
		assert.EqualValues(3, metrics.PacketsSent)
		assert.EqualValues(1, metrics.PacketsLost)

		// results getting worse
		h.AddResult(2*dur, nil)
		h.AddResult(dur, nil)
		h.AddResult(0, err)

		metrics = h.Compute()
		assert.EqualValues(100, metrics.Best)
		assert.EqualValues(200, metrics.Worst)
		assert.EqualValues(125, metrics.Mean)
		assert.InDelta(43.30127, float64(metrics.StdDev), 0.000001)
		assert.EqualValues(6, metrics.PacketsSent)
		assert.EqualValues(2, metrics.PacketsLost)

		// finally something better
		h.AddResult(0, nil)
		metrics = h.Compute()
		assert.EqualValues(0, metrics.Best)
		assert.EqualValues(200, metrics.Worst)
		assert.EqualValues(100, metrics.Mean)
		assert.InDelta(63.2455, float64(metrics.StdDev), 0.0001)
		assert.EqualValues(7, metrics.PacketsSent)
		assert.EqualValues(2, metrics.PacketsLost)
	}
}
