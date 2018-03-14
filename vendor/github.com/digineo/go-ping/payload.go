package ping

import (
	"log"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Payload represents additional data appended to outgoing ICMP Echo
// Requests.
type Payload []byte

// Resize will assign a new payload of the given size to p.
func (p *Payload) Resize(size uint16) {
	buf := make([]byte, size, size)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("error resizing payload: %v", err)
		return
	}
	*p = Payload(buf)
}
