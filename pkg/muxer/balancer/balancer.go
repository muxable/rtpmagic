package balancer

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/rtpnet"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
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

	cleanup sync.Map
}

func NewBalancedUDPConn(addr *net.UDPAddr, pollingInterval time.Duration) (*BalancedUDPConn, error) {
	ctx, cancel := context.WithCancel(context.Background())
	n := &BalancedUDPConn{
		addr:       addr,
		conns:      make(map[string]*rtpnet.CCWrapper),
		readRTPCh:  make(chan *rtp.Packet, 128),
		readRTCPCh: make(chan []rtcp.Packet, 128),
		cancel:     cancel,
		cleanup:    sync.Map{},
	}
	if err := n.bindLocalAddresses(addr); err != nil {
		return nil, err
	}
	go func() {
		ticker := time.NewTicker(pollingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := n.bindLocalAddresses(addr); err != nil {
					log.Warn().Msgf("failed to get local addresses: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return n, nil
}

// bindLocalAddresses binds the local addresses to the UDPConn.
func (n *BalancedUDPConn) bindLocalAddresses(addr *net.UDPAddr) error {
	// get the network interfaces.
	devices, err := GetLocalAddresses()
	if err != nil {
		return err
	}
	n.Lock()
	log.Printf("waiting for local addr lock")
	// add any interfaces that are not already active.
	for device := range devices {
		if _, ok := n.conns[device]; !ok {
			conn, err := DialVia(addr, device)
			if err != nil {
				log.Warn().Msgf("failed to connect to %s: %v", addr, err)
				continue
			}
			wrapped := rtpnet.NewCCWrapper(NewConnWithErrorHandler(
				conn,
				func(err error) {
					n.Lock()
					log.Printf("waiting for error lock")
					defer log.Printf("unlocking error lock")
					defer n.Unlock()
					if conn, ok := n.conns[device]; ok {
						go conn.Close()
						delete(n.conns, device)
					}
					log.Warn().Err(err).Msgf("udp error on %s", device)
				}), 1500)
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
			go conn.Close() // this can block so ignore.
			delete(n.conns, device)
			log.Info().Msgf("disconnected from %s via %s", addr, device)
		}
	}
	log.Printf("unlocking local addr")
	n.Unlock()
	// print some debugging information
	bitrate, loss := n.GetEstimatedBitrate()
	log.Debug().
		Int("Connections", len(n.conns)).
		Uint32("TotalBitrate", bitrate).
		Float64("TotalLoss", loss).
		Msg("active connections")
	keys := make([]string, 0, len(n.conns))
	for key := range n.conns {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		conn := n.conns[key]
		bitrate, loss := conn.GetEstimatedBitrate()
		log.Debug().Str("Interface", key).
			Uint32("TargetBitrate", bitrate).
			Str("RTT", conn.Sender.SenderEstimatedRoundTripTime.String()).
			Float64("LossRatio", loss).
			Float64("ECNRatio", conn.Receiver.EstimatedPacketECNMarkingRatio).
			Msg("active connection")
	}
	return nil
}

func readRTPLoop(conn *rtpnet.CCWrapper, readCh chan *rtp.Packet) {
	for {
		p, err := conn.ReadRTP()
		if err != nil {
			log.Warn().Msgf("failed to read: %v", err)
			return
		}
		readCh <- p
	}
}

func readRTCPLoop(conn *rtpnet.CCWrapper, readCh chan []rtcp.Packet) {
	for {
		pkts, err := conn.ReadRTCP()
		if err != nil {
			log.Warn().Msgf("failed to read: %v", err)
			return
		}
		readCh <- pkts
	}
}

// ReadRTP reads from the read channel.
func (n *BalancedUDPConn) ReadRTP() (*rtp.Packet, error) {
	p, ok := <-n.readRTPCh
	if !ok {
		return nil, io.EOF
	}
	return p, nil
}

// ReadRTCP reads an RTCP packet.
func (n *BalancedUDPConn) ReadRTCP() ([]rtcp.Packet, error) {
	p, ok := <-n.readRTCPCh
	if !ok {
		return nil, io.EOF
	}
	return p, nil
}

func (n *BalancedUDPConn) randomConn() *rtpnet.CCWrapper {
	bitrates := make(map[string]uint32)
	total := uint32(0)
	for key, conn := range n.conns {
		bitrates[key], _ = conn.GetEstimatedBitrate()
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
func (n *BalancedUDPConn) WriteRTP(p *rtp.Packet) error {
	n.RLock()
	defer n.RUnlock()

	if conn := n.randomConn(); conn != nil {
		return conn.WriteRTP(p)
	}
	return errNoConnection
}

// WriteRTCP writes an RTCP packet.
func (n *BalancedUDPConn) WriteRTCP(pkts []rtcp.Packet) error {
	n.RLock()
	defer n.RUnlock()

	if conn := n.randomConn(); conn != nil {
		return conn.WriteRTCP(pkts)
	}
	return errNoConnection
}

// GetEstimatedBitrate gets the estimated bitrate of the sender.
func (n *BalancedUDPConn) GetEstimatedBitrate() (uint32, float64) {
	n.RLock()
	defer n.RUnlock()

	totalBitrate := uint32(0)
	totalPacketLossRate := float64(0)
	for _, conn := range n.conns {
		bitrate, loss := conn.GetEstimatedBitrate()
		totalBitrate += bitrate
		totalPacketLossRate += loss * float64(bitrate)
	}
	return totalBitrate, totalPacketLossRate / float64(totalBitrate)
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
