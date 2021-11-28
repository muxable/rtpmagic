package netsim

import (
	"net"

	"github.com/juju/ratelimit"
)

type ThrottledConn struct {
	conn  *net.UDPConn
	read  *ratelimit.Bucket
	write *ratelimit.Bucket
}

func NewThrottledConn(conn *net.UDPConn, readLimit, writeLimit int64) *ThrottledConn {
	var read *ratelimit.Bucket
	if readLimit > 0 {
		read = ratelimit.NewBucketWithRate(float64(readLimit), readLimit)
	}
	var write *ratelimit.Bucket
	if writeLimit > 0 {
		write = ratelimit.NewBucketWithRate(float64(writeLimit), writeLimit)
	}
	return &ThrottledConn{conn, read, write}
}

func (c *ThrottledConn) Read(b []byte) (int, error) {
	n, err := c.conn.Read(b)
	if c.read != nil {
		c.read.Wait(int64(n))
	}
	return n, err
}

func (c *ThrottledConn) Write(b []byte) (int, error) {
	if c.write != nil {
		c.write.Wait(int64(len(b)))
	}
	return c.conn.Write(b)
}

// Close closes the parent connection.
func (c *ThrottledConn) Close() error {
	return c.conn.Close()
}
