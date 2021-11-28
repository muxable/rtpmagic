package balancer

import (
	"io"
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
	conns      *ConnectionMap
	seq        map[*net.UDPConn]uint16
	readRTPCh  chan *rtp.Packet
	readRTCPCh chan []rtcp.Packet
	ticker     *time.Ticker
}

const defaultHdrExtID = 5

func NewBalancedUDPConn(addr *net.UDPAddr, pollingInterval time.Duration) (*BalancedUDPConn, error) {
	ticker := time.NewTicker(pollingInterval)
	n := &BalancedUDPConn{
		addr:       addr,
		conns:      NewConnectionMap(),
		readRTPCh:  make(chan *rtp.Packet, 128),
		readRTCPCh: make(chan []rtcp.Packet, 128),
		ticker:     ticker,
	}
	go func() {
		for ok := true; ok; _, ok = <-ticker.C {
			// get the network interfaces.
			devices, err := GetLocalAddresses()
			if err != nil {
				log.Warn().Msgf("failed to get local addresses: %v", err)
				continue
			}
			// add any interfaces that are not already active.
			for device := range devices {
				if !n.conns.Has(device) {
					conn, err := net.DialUDP("udp", nil, addr)
					if err != nil {
						log.Warn().Msgf("failed to connect to %s: %v", addr, err)
					}
					wrapped := rtpnet.NewCCWrapper(conn, 1500)
					go readRTPLoop(wrapped, n.readRTPCh)
					go readRTCPLoop(wrapped, n.readRTCPCh)
					n.conns.Set(device, wrapped)
					log.Info().Msgf("connected to %s via %s", addr, device)
				}
			}
			// remove any interfaces that are no longer active.
			for device, conn := range n.conns.Items() {
				if _, ok := devices[device]; !ok {
					// remove this interface.
					if err := conn.Close(); err != nil {
						log.Warn().Msgf("failed to close connection: %v", err)
						continue
					}
					n.conns.Remove(device)
					log.Info().Msgf("disconnected from %s via %s", addr, device)
				}
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

// WriteRTP writes an RTP packet.
func (n *BalancedUDPConn) WriteRTP(p *rtp.Packet) (int, error) {
	n.RLock()
	defer n.RUnlock()

	_, conn := n.conns.Random()
	return conn.WriteRTP(p)
}

// WriteRTCP writes an RTCP packet.
func (n *BalancedUDPConn) WriteRTCP(pkts []rtcp.Packet) (int, error) {
	n.RLock()
	defer n.RUnlock()

	_, conn := n.conns.Random()
	return conn.WriteRTCP(pkts)
}

// GetEstimatedBitrate gets the estimated bitrate of the sender.
func (n *BalancedUDPConn) GetEstimatedBitrate() uint32 {
	n.RLock()
	defer n.RUnlock()

	total := uint32(0)
	for _, conn := range n.conns.Items() {
		total += conn.GetEstimatedBitrate()
	}
	return total
}

// Close closes all active connections.
func (n *BalancedUDPConn) Close() error {
	n.Lock()
	defer n.Unlock()
	close(n.readRTPCh)
	n.ticker.Stop()
	n.conns.Close()
	return nil
}
