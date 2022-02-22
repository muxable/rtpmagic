package balancer

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

type UDPConnWithErrorHandler struct {
	*net.UDPConn

	onError   func(error)
	errorOnce sync.Once

	lastReceived time.Time
	cancel       context.CancelFunc
}

var errConnectionTimeout = errors.New("connection timeout")

func NewUDPConnWithErrorHandler(conn *net.UDPConn, onError func(error)) *UDPConnWithErrorHandler {
	ctx, cancel := context.WithCancel(context.Background())
	c := &UDPConnWithErrorHandler{
		UDPConn:      conn,
		onError:      onError,
		lastReceived: time.Now(),
		cancel:       cancel,
	}
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if time.Since(c.lastReceived) > time.Second*10 {
					c.errorOnce.Do(func() {
						go c.onError(errConnectionTimeout)
					})
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return c
}

func (c *UDPConnWithErrorHandler) WriteTo(b []byte, addr net.Addr) (int, error) {
	n, err := c.UDPConn.WriteTo(b, addr)
	if err != nil {
		c.errorOnce.Do(func() {
			go c.onError(err)
		})
	}
	return n, err
}

func (c *UDPConnWithErrorHandler) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.UDPConn.ReadFrom(b)
	if err != nil {
		c.errorOnce.Do(func() {
			go c.onError(err)
		})
	}
	c.lastReceived = time.Now()
	return n, addr, err
}

func (c *UDPConnWithErrorHandler) Close() error {
	c.cancel()
	return c.UDPConn.Close()
}

var _ net.PacketConn = (*UDPConnWithErrorHandler)(nil)