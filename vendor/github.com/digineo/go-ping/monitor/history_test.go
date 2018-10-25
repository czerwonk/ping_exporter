package monitor

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkAddResult(b *testing.B) {
	h := NewHistory(8)
	for i := 0; i < b.N; i++ {
		h.AddResult(time.Duration(i), nil) // 1 allocc
	}
}

func BenchmarkCompute(b *testing.B) {
	h := NewHistory(8)
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
		h := NewHistory(4)
		assert.Nil(h.Compute())
	}

	{ // one failed entry
		h := NewHistory(4)
		h.AddResult(2, err)

		metrics := h.Compute()
		assert.EqualValues(1, metrics.PacketsSent)
		assert.EqualValues(1, metrics.PacketsLost)
		assert.EqualValues(0, metrics.Best)
		assert.EqualValues(0, metrics.Worst)
		assert.True(math.IsNaN(float64(metrics.Mean)))
		assert.True(math.IsNaN(float64(metrics.StdDev)))
	}

	{ // populate with 5 entries
		h := NewHistory(8)
		h.AddResult(0, nil)
		h.AddResult(dur, nil)
		h.AddResult(dur, nil)
		h.AddResult(0, err)
		h.AddResult(dur, nil)

		assert.Equal(h.count, 5)
		assert.EqualValues(1, h.Compute().PacketsLost)
	}

	{
		// test zero variance
		h := NewHistory(8)
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

func TestHistoryCapacity(t *testing.T) {
	assert := assert.New(t)
	err := fmt.Errorf("i/o timeout")

	h := NewHistory(3)
	assert.Equal(h.count, 0)
	h.AddResult(1, nil)
	h.AddResult(2, err)
	assert.Equal(h.count, 2)
	assert.Equal(h.position, 2)
	h.AddResult(1, nil)
	assert.Equal(h.count, 3)
	assert.Equal(h.position, 0)

	h.AddResult(0, nil)
	assert.Equal(h.count, 3)
	assert.Equal(h.position, 1)
	assert.EqualValues(1, h.Compute().PacketsLost)

	// overwrite lost packet result
	h.AddResult(0, nil)
	assert.EqualValues(0, h.Compute().PacketsLost)

	// clear
	h.ComputeAndClear()
	assert.Equal(h.count, 0)
	assert.Equal(h.position, 0)
}
