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

	"github.com/muxable/rtpmagic/pkg/muxer/balancer/bind"
	"github.com/muxable/rtpmagic/pkg/muxer/rtpnet"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

type readResult struct {
	buf  []byte
	addr net.Addr
}

type BalancedUDPConn struct {
	sync.RWMutex

	conns  map[string]*rtpnet.CCWrapper
	readCh chan *readResult
	cancel context.CancelFunc

	cleanup sync.Map
}

func NewBalancedUDPConn(pollingInterval time.Duration) (*BalancedUDPConn, error) {
	ctx, cancel := context.WithCancel(context.Background())
	n := &BalancedUDPConn{
		conns:   make(map[string]*rtpnet.CCWrapper),
		readCh:  make(chan *readResult),
		cancel:  cancel,
		cleanup: sync.Map{},
	}
	if err := n.bindLocalAddresses(); err != nil {
		return nil, err
	}
	go func() {
		ticker := time.NewTicker(pollingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := n.bindLocalAddresses(); err != nil {
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
func (n *BalancedUDPConn) bindLocalAddresses() error {
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
			conn, err := net.ListenUDP("udp", &net.UDPAddr{})
			if err != nil {
				log.Warn().Msgf("failed to connect %v", err)
				continue
			}
			if err := bind.BindToDevice(conn, device); err != nil {
				log.Printf("error binding to device %v %v", device, err)
				continue
			}
			wrapped := rtpnet.NewCCWrapper(conn)
			go func() {
				buf := make([]byte, 1500)
				for {
					if err := wrapped.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
						log.Warn().Msgf("failed to set read deadline: %v", err)
					}
					m, addr, err := wrapped.ReadFrom(buf)
					if err != nil {
						n.Lock()
						if conn, ok := n.conns[device]; ok {
							go conn.Close()
							delete(n.conns, device)
						}
						n.Unlock()
						log.Warn().Err(err).Msgf("udp error on %s", device)
					}
					n.readCh <- &readResult{
						buf:  buf[:m],
						addr: addr,
					}
				}
			}()
			n.conns[device] = wrapped
			log.Info().Msgf("connected via %s", device)
		}
	}
	// remove any interfaces that are no longer active.
	for device, conn := range n.conns {
		if _, ok := devices[device]; !ok {
			// remove this interface.
			go conn.Close() // this can block so ignore.
			delete(n.conns, device)
			log.Info().Msgf("disconnected via %s", device)
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

func (n *BalancedUDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	res := <-n.readCh
	if len(b) < len(res.buf) {
		return 0, nil, io.ErrShortBuffer
	}
	return len(res.buf), res.addr, nil
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
func (n *BalancedUDPConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	n.RLock()
	defer n.RUnlock()

	if conn := n.randomConn(); conn != nil {
		return conn.WriteTo(b, addr)
	}
	return 0, errNoConnection
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
	close(n.readCh)
	n.cancel()
	for _, conn := range n.conns {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (n *BalancedUDPConn) LocalAddr() net.Addr {
	return n
}

func (n *BalancedUDPConn) Network() string {
	return "udp"
}

func (n *BalancedUDPConn) String() string {
	return "udp-cc-multicast"
}

func (n *BalancedUDPConn) SetDeadline(t time.Time) error {
	n.Lock()
	defer n.Unlock()

	var err error
	for _, conn := range n.conns {
		err = multierr.Append(err, conn.SetDeadline(t))
	}
	return err
}

func (n *BalancedUDPConn) SetReadDeadline(t time.Time) error {
	n.Lock()
	defer n.Unlock()

	var err error
	for _, conn := range n.conns {
		err = multierr.Append(err, conn.SetReadDeadline(t))
	}
	return err
}

func (n *BalancedUDPConn) SetWriteDeadline(t time.Time) error {
	n.Lock()
	defer n.Unlock()

	var err error
	for _, conn := range n.conns {
		err = multierr.Append(err, conn.SetWriteDeadline(t))
	}
	return err
}

var _ net.PacketConn = (*BalancedUDPConn)(nil)
