package test

import (
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type NetSimUDPConnConfig struct {
	NumConnections         uint32
	ReconnectionLikelihood float64
	ReconnectionPeriod     time.Duration
	ReconnectionDelay      time.Duration
	DropRate               float64
	PacketDelayVariance    time.Duration
}

type NetSimUDPConn struct {
	sync.Mutex

	activeConnections []*net.UDPConn
	closed            bool
	config            NetSimUDPConnConfig
}

func NewNetSimUDPConn(addr *net.UDPAddr, config NetSimUDPConnConfig) *NetSimUDPConn {
	n := &NetSimUDPConn{
		activeConnections: make([]*net.UDPConn, config.NumConnections),
		config:            config,
	}
	// reconnection loop.
	go func() {
		for {
			time.Sleep(config.ReconnectionPeriod)

			n.Lock()
			if n.closed {
				return
			}

			if rand.Float64() < config.ReconnectionLikelihood {
				// drop a connection randomly.
				i := rand.Intn(len(n.activeConnections))
				if err := n.activeConnections[i].Close(); err != nil {
					log.Warn().Msgf("failed to close connection: %v", err)
					continue
				}

				// wait to reconnect.
				time.Sleep(config.ReconnectionDelay)

				// then create a new connection.
				c, err := net.DialUDP("udp", nil, addr)
				if err != nil {
					log.Warn().Msgf("failed to reconnect to %s: %v", addr.String(), err)
					continue
				}
				n.activeConnections[i] = c
			}

			n.Unlock()
		}
	}()
	return n
}

func (n *NetSimUDPConn) Write(data []byte) (int, error) {
	if rand.Float64() < n.config.DropRate {
		return len(data), nil
	}
	dup := make([]byte, len(data))
	copy(dup, data)
	go func() {
		time.Sleep(time.Duration(rand.ExpFloat64() / float64(n.config.PacketDelayVariance)))
		// write to a random socket.
		i := rand.Intn(len(n.activeConnections))
		n.activeConnections[i].Write(data)
	}()
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
