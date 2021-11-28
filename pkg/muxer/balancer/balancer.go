package balancer

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/muxer/rtpnet"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type BalancedUDPConn struct {
	sync.RWMutex

	rtpio.RTPReadWriteCloser
	rtpio.RTCPReadWriteCloser

	addr       *net.UDPAddr
	conns      map[string]*rtpnet.CCWrapper
	readRTPCh  chan *rtp.Packet
	readRTCPCh chan []rtcp.Packet
	cancel     context.CancelFunc
}

func NewBalancedUDPConn(addr *net.UDPAddr, pollingInterval time.Duration) (*BalancedUDPConn, error) {
	ctx, cancel := context.WithCancel(context.Background())
	n := &BalancedUDPConn{
		addr:       addr,
		conns:      make(map[string]*rtpnet.CCWrapper),
		readRTPCh:  make(chan *rtp.Packet, 128),
		readRTCPCh: make(chan []rtcp.Packet, 128),
		cancel:     cancel,
	}
	go func() {
		ticker := time.NewTicker(pollingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// get the network interfaces.
				devices, err := GetLocalAddresses()
				if err != nil {
					log.Warn().Msgf("failed to get local addresses: %v", err)
					continue
				}
				n.Lock()
				// add any interfaces that are not already active.
				for device := range devices {
					if _, ok := n.conns[device]; !ok {
						conn, err := net.DialUDP("udp", nil, addr)
						if err != nil {
							log.Warn().Msgf("failed to connect to %s: %v", addr, err)
						}
						wrapped := rtpnet.NewCCWrapper(conn, 1500)
						go readRTPLoop(wrapped, n.readRTPCh)
						go readRTCPLoop(wrapped, n.readRTCPCh)
						n.conns[device] = wrapped
						log.Info().Msgf("connected to %s via %s", addr, device)
					}
				}
				// remove any interfaces that are no longer active.
				for device, conn := range n.conns {
					if _, ok := devices[device]; !ok {
						// remove this interface.
						if err := conn.Close(); err != nil {
							log.Warn().Msgf("failed to close connection: %v", err)
							continue
						}
						delete(n.conns, device)
						log.Info().Msgf("disconnected from %s via %s", addr, device)
					}
				}
				n.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(pollingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n.RLock()
				// print some debugging information
				log.Debug().Int("Connections", len(n.conns)).Msg("active connections")
				for key, conn := range n.conns {
					log.Debug().Str("Interface", key).Uint32("Bitrate", conn.GetEstimatedBitrate()).Msg("active connection")
				}
				n.RUnlock()
			case <-ctx.Done():
				return
			}
		}
	}()
	return n, nil
}

func readRTPLoop(conn *rtpnet.CCWrapper, readCh chan *rtp.Packet) {
	for {
		p := &rtp.Packet{}
		if _, err := conn.ReadRTP(p); err != nil {
			log.Warn().Msgf("failed to read: %v", err)
			return
		}
		readCh <- p
	}
}

func readRTCPLoop(conn *rtpnet.CCWrapper, readCh chan []rtcp.Packet) {
	for {
		pkts := make([]rtcp.Packet, 16)
		if _, err := conn.ReadRTCP(pkts); err != nil {
			log.Warn().Msgf("failed to read: %v", err)
			return
		}
		readCh <- pkts
	}
}

// ReadRTP reads from the read channel.
func (n *BalancedUDPConn) ReadRTP(p *rtp.Packet) (int, error) {
	q, ok := <-n.readRTPCh
	if !ok {
		return 0, io.EOF
	}
	*p = *q
	return q.MarshalSize(), nil
}

// ReadRTCP reads an RTCP packet.
func (n *BalancedUDPConn) ReadRTCP(pkts []rtcp.Packet) (int, error) {
	q, ok := <-n.readRTCPCh
	if !ok {
		return 0, io.EOF
	}
	return copy(pkts, q), nil
}

func (n *BalancedUDPConn) randomConn() *rtpnet.CCWrapper {
	bitrates := make(map[string]uint32)
	total := uint32(0)
	for key, conn := range n.conns {
		bitrates[key] = conn.GetEstimatedBitrate()
		total += bitrates[key]
	}
	if total == 0 {
		return nil
	}
	index := rand.Intn(int(total))
	for key, bitrate := range bitrates {
		if index < int(bitrate) {
			return n.conns[key]
		}
		index -= int(bitrate)
	}
	return nil
}

var errNoConnection = errors.New("no connection available")

// WriteRTP writes an RTP packet.
func (n *BalancedUDPConn) WriteRTP(p *rtp.Packet) (int, error) {
	n.RLock()
	defer n.RUnlock()

	if conn := n.randomConn(); conn != nil {
		return conn.WriteRTP(p)
	}
	return 0, errNoConnection
}

// WriteRTCP writes an RTCP packet.
func (n *BalancedUDPConn) WriteRTCP(pkts []rtcp.Packet) (int, error) {
	n.RLock()
	defer n.RUnlock()

	if conn := n.randomConn(); conn != nil {
		return conn.WriteRTCP(pkts)
	}
	return 0, errNoConnection
}

// GetEstimatedBitrate gets the estimated bitrate of the sender.
func (n *BalancedUDPConn) GetEstimatedBitrate() uint32 {
	n.RLock()
	defer n.RUnlock()

	total := uint32(0)
	for _, conn := range n.conns {
		total += conn.GetEstimatedBitrate()
	}
	return total
}

// Close closes all active connections.
func (n *BalancedUDPConn) Close() error {
	n.Lock()
	defer n.Unlock()
	close(n.readRTPCh)
	n.cancel()
	for _, conn := range n.conns {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}