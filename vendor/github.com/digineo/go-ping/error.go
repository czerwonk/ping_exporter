package ping

import "errors"

var (
	errClosed   = errors.New("pinger closed")
	errNotBound = errors.New("need at least one bind address")
)

// timeoutError implements the net.Error interface. Originally taken from
// https://github.com/golang/go/blob/release-branch.go1.8/src/net/net.go#L505-L509
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
