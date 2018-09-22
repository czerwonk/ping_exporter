package ping

import (
	"os"
	"sync"

	"golang.org/x/net/icmp"
)

const (
	// ProtocolICMP is the number of the Internet Control Message Protocol
	// (see golang.org/x/net/internal/iana.ProtocolICMP)
	ProtocolICMP = 1

	// ProtocolICMPv6 is the IPv6 Next Header value for ICMPv6
	// see golang.org/x/net/internal/iana.ProtocolIPv6ICMP
	ProtocolICMPv6 = 58
)

// sequence number for this process
var sequence uint32

// Pinger is a instance for ICMP echo requests
type Pinger struct {
	LogUnexpectedPackets bool // increases log verbosity

	payload   Payload
	payloadMu sync.RWMutex

	requests map[uint16]request // currently running requests
	mtx      sync.RWMutex       // lock for the requests map
	id       uint16
	conn4    *icmp.PacketConn
	conn6    *icmp.PacketConn
	write4   sync.Mutex // lock for conn4.WriteTo
	write6   sync.Mutex // lock for conn6.WriteTo
	wg       sync.WaitGroup
}

// New creates a new Pinger. This will open the raw socket and start the
// receiving logic. You'll need to call Close() to cleanup.
func New(bind4, bind6 string) (*Pinger, error) {
	// open sockets
	conn4, err := connectICMP("ip4:icmp", bind4)
	if err != nil {
		return nil, err
	}

	conn6, err := connectICMP("ip6:ipv6-icmp", bind6)
	if err != nil {
		if conn4 != nil {
			conn4.Close()
		}
		return nil, err
	}

	if conn4 == nil && conn6 == nil {
		return nil, errNotBound
	}

	pinger := Pinger{
		conn4:    conn4,
		conn6:    conn6,
		id:       uint16(os.Getpid()),
		requests: make(map[uint16]request),
	}
	pinger.SetPayloadSize(56)

	if conn4 != nil {
		pinger.wg.Add(1)
		go pinger.receiver(ProtocolICMP, pinger.conn4)
	}
	if conn6 != nil {
		pinger.wg.Add(1)
		go pinger.receiver(ProtocolICMPv6, pinger.conn6)
	}

	return &pinger, nil
}

// Close will close the ICMP socket.
func (pinger *Pinger) Close() {
	pinger.close(pinger.conn4)
	pinger.close(pinger.conn6)
	pinger.wg.Wait()
}

// connectICMP opens a new ICMP connection, if network and address are not empty.
func connectICMP(network, address string) (*icmp.PacketConn, error) {
	if network == "" || address == "" {
		return nil, nil
	}

	return icmp.ListenPacket(network, address)
}

func (pinger *Pinger) close(conn *icmp.PacketConn) {
	if conn != nil {
		conn.Close()
	}
}

func (pinger *Pinger) removeRequest(seq uint16) {
	pinger.mtx.Lock()
	delete(pinger.requests, seq)
	pinger.mtx.Unlock()
}

// SetPayloadSize resizes additional payload data to the given size. The
// payload will subsequently be appended to outgoing ICMP Echo Requests.
//
// The default payload size is 56, resulting in 64 bytes for the ICMP packet.
func (pinger *Pinger) SetPayloadSize(size uint16) {
	pinger.payloadMu.Lock()
	pinger.payload.Resize(size)
	pinger.payloadMu.Unlock()
}

// SetPayload allows you to overwrite the current payload with your own data.
func (pinger *Pinger) SetPayload(data []byte) {
	pinger.payloadMu.Lock()
	pinger.payload = Payload(data)
	pinger.payloadMu.Unlock()
}

// PayloadSize retrieves the current payload size.
func (pinger *Pinger) PayloadSize() uint16 {
	pinger.payloadMu.RLock()
	defer pinger.payloadMu.RUnlock()
	return uint16(len(pinger.payload))
}
