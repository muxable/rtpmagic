package balancer

import (
	"net"
	"sync"
)

type UDPConnWithErrorHandler struct {
	*net.UDPConn

	onError func(error)
	errorOnce sync.Once
}

func (c *UDPConnWithErrorHandler) Write(b []byte) (int, error) {
	n, err := c.UDPConn.Write(b)
	if err != nil {
		c.errorOnce.Do(func() {
			go c.onError(err)
		})
	}
	return n, err
}

func (c *UDPConnWithErrorHandler) Read(b []byte) (int, error) {
	n, err := c.UDPConn.Read(b)
	if err != nil {
		c.errorOnce.Do(func() {
			go c.onError(err)
		})
	}
	return n, err
}

func (c *UDPConnWithErrorHandler) Close() error {
	return c.UDPConn.Close()
}
