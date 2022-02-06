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
	closed            bool
	configs           []*ConnectionState
	readRTPCh         chan *rtp.Packet
	readRTCPCh        chan []rtcp.Packet
}

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
			p, err := conn.ReadRTP()
			if err != nil {
				log.Warn().Msgf("failed to read: %v", err)
				return
			}
			n.readRTPCh <- p
		}
	}()
	go func() {
		for {
			p, err := conn.ReadRTCP()
			if err != nil {
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
func (n *NetSimUDPConn) ReadRTP() (*rtp.Packet, error) {
	p, ok := <-n.readRTPCh
	if !ok {
		return nil, io.EOF
	}
	return p, nil
}

// ReadRTCP reads an RTCP packet.
func (n *NetSimUDPConn) ReadRTCP() ([]rtcp.Packet, error) {
	p, ok := <-n.readRTCPCh
	if !ok {
		return nil, io.EOF
	}
	return p, nil
}

// WriteRTP writes an RTP packet.
func (n *NetSimUDPConn) WriteRTP(p *rtp.Packet) error {
	i := rand.Intn(len(n.activeConnections))
	config := n.configs[i]
	writeCount := 1
	if rand.Float64() < config.DropRate {
		// this packet got dropped.
		return nil
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
			if err := conn.WriteRTP(p); err != nil {
				log.Warn().Msgf("failed to write: %v", err)
			}
		}
	}()
	return nil
}

// WriteRTCP writes an RTCP packet.
func (n *NetSimUDPConn) WriteRTCP(pkts []rtcp.Packet) error {
	i := rand.Intn(len(n.activeConnections))
	config := n.configs[i]
	writeCount := 1
	if rand.Float64() < config.DropRate {
		// this packet got dropped.
		return nil
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
			if err := conn.WriteRTCP(pkts); err != nil {
				log.Warn().Msgf("failed to write: %v", err)
			}
		}
	}()
	return nil
}

func (n *NetSimUDPConn) GetEstimatedBitrate() (uint32, float64) {
	totalBitrate := uint32(0)
	totalPacketLossRate := float64(0)
	for _, conn := range n.activeConnections {
		bitrate, loss := conn.GetEstimatedBitrate()
		totalBitrate += bitrate
		totalPacketLossRate += loss * float64(bitrate)
	}
	return totalBitrate, totalPacketLossRate / float64(totalBitrate)
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
