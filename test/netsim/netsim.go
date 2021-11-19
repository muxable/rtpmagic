package netsim

import (
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type ConnectionState struct {
	ReconnectionLikelihood float64
	ReconnectionPeriod     time.Duration
	ReconnectionDelay      time.Duration
	DropRate               float64
	DuplicateRate          float64
	LatencyMean            time.Duration
	LatencyVariance        time.Duration
}

type NetSimUDPConn struct {
	sync.RWMutex

	addr              *net.UDPAddr
	activeConnections []*net.UDPConn
	closed            bool
	configs           []*ConnectionState
	readChan          chan []byte
}

func NewNetSimUDPConn(addr *net.UDPAddr, configs []*ConnectionState) (*NetSimUDPConn, error) {
	n := &NetSimUDPConn{
		addr:              addr,
		activeConnections: make([]*net.UDPConn, len(configs)),
		configs:           configs,
		readChan:          make(chan []byte, 128),
	}
	// reconnection loop.
	for i, config := range configs {
		n.reconnect(i)
		go func(config *ConnectionState) {
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
					n.reconnect(i)
				}

				n.Unlock()
			}
		}(config)
	}
	return n, nil
}

// reconnect reconnects a specific index.
func (n *NetSimUDPConn) reconnect(i int) error {
	c, err := net.DialUDP("udp", nil, n.addr)
	if err != nil {
		log.Warn().Msgf("failed to reconnect to %s: %v", n.addr, err)
		return err
	}
	go func() {
		for {
			buf := make([]byte, 1500)
			len, err := c.Read(buf)
			if err != nil {
				log.Warn().Msgf("failed to read: %v", err)
				return
			}
			n.readChan <- buf[:len]
		}
	}()
	n.activeConnections[i] = c
	return nil
}

// Read reads from the read channel.
func (n *NetSimUDPConn) Read(b []byte) (int, error) {
	buf, ok := <-n.readChan
	if !ok {
		return 0, io.EOF
	}
	copy(b, buf)
	return len(buf), nil
}

func (n *NetSimUDPConn) Write(data []byte) (int, error) {
	i := rand.Intn(len(n.activeConnections))
	config := n.configs[i]
	writeCount := 1
	if rand.Float64() < config.DropRate {
		// this packet got dropped.
		p := &rtp.Packet{}
		p.Unmarshal(data)
		return len(data), nil
	}
	if rand.Float64() < config.DuplicateRate {
		// this packet got duplicated.
		writeCount = 2
	}
	dup := make([]byte, len(data))
	copy(dup, data)
	go func() {
		time.Sleep(time.Duration(rand.NormFloat64()*1000000)*time.Microsecond*config.LatencyVariance + config.LatencyMean)
		n.RLock()
		for j := 0; j < writeCount; j++ {
			if _, err := n.activeConnections[i].Write(data); err != nil {
				log.Warn().Msgf("failed to write: %v", err)
			}
		}
		n.RUnlock()
	}()
	return len(data), nil
}

// Close closes all active connections.
func (n *NetSimUDPConn) Close() error {
	n.Lock()
	defer n.Unlock()
	close(n.readChan)
	for _, conn := range n.activeConnections {
		conn.Close()
	}
	n.closed = true
	return nil
}
