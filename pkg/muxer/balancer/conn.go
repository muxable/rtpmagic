package balancer

import (
	"net"
)

type UDPConnWithErrorHandler struct {
	*net.UDPConn

	onError func(error)

	errored bool
}

func (c *UDPConnWithErrorHandler) Write(b []byte) (int, error) {
	n, err := c.UDPConn.Write(b)
	if err != nil {
		if c.errored {
			return n, err
		}
		defer c.onError(err)
		c.errored = true
	}
	return n, err
}

func (c *UDPConnWithErrorHandler) Read(b []byte) (int, error) {
	n, err := c.UDPConn.Read(b)
	if err != nil {
		if c.errored {
			return n, err
		}
		defer c.onError(err)
		c.errored = true
	}
	return n, err
}

func (c *UDPConnWithErrorHandler) Close() error {
	return c.UDPConn.Close()
}
