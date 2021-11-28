package netsim

import (
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/rtpnet"
	"github.com/pion/rtcp"
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
	MaxUploadBitrate       int64
	MaxDownloadBitrate     int64
}

type NetSimUDPConn struct {
	sync.RWMutex

	addr              *net.UDPAddr
	activeConnections []*rtpnet.CCWrapper
	seq               map[*ThrottledConn]uint16
	closed            bool
	configs           []*ConnectionState
	readRTPCh         chan *rtp.Packet
	readRTCPCh        chan []rtcp.Packet
}

const defaultHdrExtID = 5

func NewNetSimUDPConn(addr *net.UDPAddr, configs []*ConnectionState) (*NetSimUDPConn, error) {
	n := &NetSimUDPConn{
		addr:              addr,
		activeConnections: make([]*rtpnet.CCWrapper, len(configs)),
		configs:           configs,
		readRTPCh:         make(chan *rtp.Packet, 128),
		readRTCPCh:        make(chan []rtcp.Packet, 128),
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
	tc := NewThrottledConn(c, n.configs[i].MaxUploadBitrate, n.configs[i].MaxDownloadBitrate)
	conn := rtpnet.NewCCWrapper(tc, 1500)
	go func() {
		for {
			p := &rtp.Packet{}
			if _, err := conn.ReadRTP(p); err != nil {
				log.Warn().Msgf("failed to read: %v", err)
				return
			}
			n.readRTPCh <- p
		}
	}()
	go func() {
		for {
			p := make([]rtcp.Packet, 16)
			if _, err := conn.ReadRTCP(p); err != nil {
				log.Warn().Msgf("failed to read: %v", err)
				return
			}
			n.readRTCPCh <- p
		}
	}()
	n.activeConnections[i] = conn
	return nil
}

// ReadRTP reads from the read channel.
func (n *NetSimUDPConn) ReadRTP(p *rtp.Packet) (int, error) {
	q, ok := <-n.readRTPCh
	if !ok {
		return 0, io.EOF
	}
	*p = *q
	return p.MarshalSize(), nil
}

// ReadRTCP reads an RTCP packet.
func (n *NetSimUDPConn) ReadRTCP(p []rtcp.Packet) (int, error) {
	q, ok := <-n.readRTCPCh
	if !ok {
		return 0, io.EOF
	}
	return copy(p, q), nil
}

// WriteRTP writes an RTP packet.
func (n *NetSimUDPConn) WriteRTP(p *rtp.Packet) (int, error) {
	i := rand.Intn(len(n.activeConnections))
	config := n.configs[i]
	writeCount := 1
	if rand.Float64() < config.DropRate {
		// this packet got dropped.
		return p.MarshalSize(), nil
	}
	if rand.Float64() < config.DuplicateRate {
		// this packet got duplicated.
		writeCount = 2
	}
	conn := n.activeConnections[i]
	go func() {
		time.Sleep(time.Duration(rand.NormFloat64()*1000000)*time.Microsecond*config.LatencyVariance + config.LatencyMean)
		n.RLock()
		defer n.RUnlock()
		for j := 0; j < writeCount; j++ {
			if _, err := conn.WriteRTP(p); err != nil {
				log.Warn().Msgf("failed to write: %v", err)
			}
		}
	}()
	return p.MarshalSize(), nil
}

// WriteRTCP writes an RTCP packet.
func (n *NetSimUDPConn) WriteRTCP(pkts []rtcp.Packet) (int, error) {
	i := rand.Intn(len(n.activeConnections))
	config := n.configs[i]
	writeCount := 1
	if rand.Float64() < config.DropRate {
		// this packet got dropped.
		return len(pkts), nil
	}
	if rand.Float64() < config.DuplicateRate {
		// this packet got duplicated.
		writeCount = 2
	}
	conn := n.activeConnections[i]
	go func() {
		time.Sleep(time.Duration(rand.NormFloat64()*1000000)*time.Microsecond*config.LatencyVariance + config.LatencyMean)
		n.RLock()
		defer n.RUnlock()
		for j := 0; j < writeCount; j++ {
			if _, err := conn.WriteRTCP(pkts); err != nil {
				log.Warn().Msgf("failed to write: %v", err)
			}
		}
	}()
	return len(pkts), nil
}

func (n *NetSimUDPConn) GetEstimatedBitrate() uint32 {
	total := uint32(0)
	for _, conn := range n.activeConnections {
		total += conn.GetEstimatedBitrate()
	}
	return total
}

// Close closes all active connections.
func (n *NetSimUDPConn) Close() error {
	n.Lock()
	defer n.Unlock()
	close(n.readRTPCh)
	for _, conn := range n.activeConnections {
		conn.Close()
	}
	n.closed = true
	return nil
}
