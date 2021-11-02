package test

import (
	"net"
	"sync"
	"time"
)

type NetSimUDPConn struct {
	sync.Mutex

	addr              net.UDPAddr
	activeConnections []net.UDPConn
	closed            bool

	NumConnections     uint32
	ReconnectionPeriod time.Duration
	ReconnectionDelay  time.Duration
	DropRate           float64
	ReorderVariance    time.Duration
}

func NewNetSimUDPConn(addr net.UDPAddr) *NetSimUDPConn {
	n := &NetSimUDPConn{
		addr:              addr,
		activeConnections: []net.UDPConn{},
	}
	go func() {
		for {
			time.Sleep(n.ReconnectionPeriod)
			n.reconnect()
		}
	}()
	return n
}

func (n *NetSimUDPConn) Write(data []byte) (int, error) {
	return len(data), nil
}

// Close closes all active connections.
func (n *NetSimUDPConn) Close() error {
	n.Lock()
	defer n.Unlock()
	for _, conn := range n.activeConnections {
		conn.Close()
	}
	n.closed = true
	return nil
}
