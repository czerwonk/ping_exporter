package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeStats(t *testing.T) {
	assert := assert.New(t)
	const (
		z  = time.Duration(0)
		ms = time.Millisecond
		µs = time.Microsecond
		ns = time.Nanosecond
	)

	testcases := []struct {
		title    string
		results  []time.Duration
		received int
		lost     int

		last   time.Duration
		best   time.Duration
		worst  time.Duration
		mean   time.Duration
		stddev time.Duration
		loss   float64
	}{
		{
			title:    "simplest case",
			results:  []time.Duration{},
			received: 0,
			last:     z, best: z, worst: z, mean: z, stddev: z,
		},
		{
			title:    "another simple case",
			results:  []time.Duration{ms},
			received: 1,
			last:     ms, best: ms, worst: ms, mean: ms, stddev: z,
		},
		{
			title:    "same as before, but sent>len(res)",
			results:  []time.Duration{ms},
			received: 3,
			last:     ms, best: ms, worst: ms, mean: ms, stddev: z,
		},
		{
			title:    "same as before, but sent<len(res)",
			results:  []time.Duration{ms, ms, 5 * ms},
			received: 2,
			last:     ms, best: ms, worst: ms, mean: ms, stddev: z,
		},
		{
			title:    "different numbers, manually calculated",
			results:  []time.Duration{ms, 2 * ms},
			received: 2,
			last:     2 * ms,
			best:     ms,
			worst:    2 * ms,
			mean:     1500 * µs,
			stddev:   500 * µs,
		},
		{
			title:    "wilder numbers",
			results:  []time.Duration{6 * ms, 2 * ms, 14 * ms, 11 * ms},
			received: 6,
			lost:     2,
			last:     2 * ms, // res[received%len]
			best:     2 * ms,
			worst:    14 * ms,
			mean:     8250 * µs, // (6000+2000+14000+11000)/4
			stddev:   4602988,   // 4602988.15988
			loss:     0.25,      // sent = 6+2
		},
		{
			title:    "verifying captured data",
			received: 50,
			lost:     7,
			loss:     0.1228, // 7 / 57

			last:   488619758,
			best:   451327200,
			worst:  492082650,
			mean:   487287379,
			stddev: 9356133,

			results: []time.Duration{
				478427841, 486727913, 489902185, 490369676, 489957386,
				490784152, 491390728, 491012043, 491313203, 489869560,
				488634310, 451590351, 480933928, 451431418, 491046095,
				492017348, 488906398, 490187284, 490733777, 490418928,
				490627269, 490710944, 491339118, 491300740, 490320794,
				489706066, 487735713, 488153523, 490988560, 490293234,
				492082650, 490784586, 488731408, 488008147, 487630508,
				490190288, 490712289, 489931645, 490608008, 490625639,
				491721463, 451327200, 491615584, 490238328, 489234608,
				488510694, 488807517, 489176334, 488981822, 488619758,
			},
		},
	}

	for i, tc := range testcases {
		h := history{received: tc.received, results: tc.results, lost: tc.lost}
		subject := h.compute()

		assert.Equal(tc.best, subject.best, "test case #%d (%s): best", i, tc.title)
		assert.Equal(tc.last, subject.last, "test case #%d (%s): last", i, tc.title)
		assert.Equal(tc.worst, subject.worst, "test case #%d (%s): worst", i, tc.title)
		assert.Equal(tc.mean, subject.mean, "test case #%d (%s): mean", i, tc.title)
		assert.Equal(tc.stddev, subject.stddev, "test case #%d (%s): stddev", i, tc.title)
		assert.Equal(tc.received+tc.lost, subject.pktSent, "test case #%d (%s): pktSent", i, tc.title)
		assert.InDelta(tc.loss, subject.pktLoss, 0.0001, "test case #%d (%s): pktLoss", i, tc.title)
	}
}
