package ping

import (
	"time"
)

// A request is a currently running ICMP echo request waiting for an answer.
type request struct {
	wait    chan struct{}
	result  error
	tStart  time.Time  // when was this packet sent?
	tFinish *time.Time // if and when was the response received?
}

// respond is responsible for finishing this request. It takes an error
// as failure reason.
func (req *request) respond(err error, tRecv *time.Time) {
	req.result = err

	// update tFinish only if no error present and value wasn't previously set
	if err == nil && tRecv != nil && req.tFinish == nil {
		req.tFinish = tRecv
	}
	close(req.wait)
}

func (req *request) roundTripTime() (time.Duration, error) {
	if req.result != nil {
		return 0, req.result
	}
	if req.tFinish == nil {
		return 0, nil
	}
	return req.tFinish.Sub(req.tStart), nil
}
