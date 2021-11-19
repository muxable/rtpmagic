package balancer

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type BalancedUDPConn struct {
	sync.RWMutex

	addr              *net.UDPAddr
	conns *ConnectionMap
	readCh            chan []byte
	ticker            *time.Ticker
}

func NewBalancedUDPConn(addr *net.UDPAddr, pollingInterval time.Duration) (*BalancedUDPConn, error) {
	ticker := time.NewTicker(pollingInterval)
	n := &BalancedUDPConn{
		addr:              addr,
		conns: NewConnectionMap(),
		readCh:            make(chan []byte, 128),
		ticker:            ticker,
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
					conn, err := DialVia(addr, device)
					if err != nil {
						log.Warn().Msgf("failed to connect to %s: %v", addr, err)
					}
					go readLoop(conn, n.readCh)
					n.conns.Set(device, conn)
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

// reconnect reconnects a specific index.
func readLoop(conn *net.UDPConn, readCh chan []byte) {
	for {
		buf := make([]byte, 1500)
		len, err := conn.Read(buf)
		if err != nil {
			log.Warn().Msgf("failed to read: %v", err)
			return
		}
		readCh <- buf[:len]
	}
}

// Read reads from the read channel.
func (n *BalancedUDPConn) Read(b []byte) (int, error) {
	buf, ok := <-n.readCh
	if !ok {
		return 0, io.EOF
	}
	copy(b, buf)
	return len(buf), nil
}

func (n *BalancedUDPConn) Write(data []byte) (int, error) {
	n.RLock()
	defer n.RUnlock()

	_, conn := n.conns.Random()
	if _, err := conn.Write(data); err != nil {
		log.Warn().Msgf("failed to write: %v", err)
	}
	return len(data), nil
}

// Close closes all active connections.
func (n *BalancedUDPConn) Close() error {
	n.Lock()
	defer n.Unlock()
	close(n.readCh)
	n.ticker.Stop()
	n.conns.Close()
	return nil
}
