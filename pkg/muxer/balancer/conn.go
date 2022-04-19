package balancer

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

type ConnWithErrorHandler struct {
	net.Conn

	onError   func(error)
	errorOnce sync.Once

	lastReceived time.Time
	cancel       context.CancelFunc
}

var errConnectionTimeout = errors.New("connection timeout")

func NewConnWithErrorHandler(conn net.Conn, onError func(error)) *ConnWithErrorHandler {
	ctx, cancel := context.WithCancel(context.Background())
	c := &ConnWithErrorHandler{
		Conn:      conn,
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

func (c *ConnWithErrorHandler) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if err != nil {
		c.errorOnce.Do(func() {
			go c.onError(err)
		})
	}
	return n, err
}

func (c *ConnWithErrorHandler) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err != nil {
		c.errorOnce.Do(func() {
			go c.onError(err)
		})
	}
	c.lastReceived = time.Now()
	return n, err
}

func (c *ConnWithErrorHandler) Close() error {
	c.cancel()
	return c.Conn.Close()
}
