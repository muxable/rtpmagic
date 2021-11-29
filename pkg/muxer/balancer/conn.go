package balancer

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type UDPConnWithRetry struct {
	sync.RWMutex
	*net.UDPConn
	to *net.UDPAddr

	lastRead time.Time
	cancel   context.CancelFunc
}

// DialUDPWithRetry monitors the connection and if no data is received after a period of time it will auto-reconnect.
func DialUDPWithRetry(to *net.UDPAddr) (*UDPConnWithRetry, error) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &UDPConnWithRetry{
		UDPConn:  nil,
		to:       to,
		lastRead: time.Now(),
		cancel:   cancel,
	}
	if err := c.dial(); err != nil {
		return nil, err
	}
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if time.Since(c.lastRead) > 15*time.Second {
					log.Warn().Msg("UDP connection timed out, reconnecting...")
					if err := c.dial(); err != nil {
						log.Error().Err(err).Msg("failed to redial")
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return c, nil
}

func (c *UDPConnWithRetry) dial() error {
	if c.UDPConn != nil {
		c.UDPConn.Close()
	}
	conn, err := net.DialUDP("udp", nil, c.to)
	if err != nil {
		return err
	}
	c.UDPConn = conn
	c.lastRead = time.Now()
	return nil
}

func (c *UDPConnWithRetry) Write(b []byte) (int, error) {
	c.RLock()
	defer c.RUnlock()

	return c.UDPConn.Write(b)
}

func (c *UDPConnWithRetry) Read(b []byte) (int, error) {
	c.Lock()
	defer c.Unlock()

	n, err := c.UDPConn.Read(b)
	if err != nil {
		return n, err
	}
	c.lastRead = time.Now()
	return n, nil
}

func (c *UDPConnWithRetry) Close() error {
	c.cancel()
	return c.UDPConn.Close()
}
