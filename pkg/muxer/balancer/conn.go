package balancer

import (
	"net"
	"time"
)

type UDPConnWithErrorHandler struct {
        *net.UDPConn

        onError func()
}

// DialUDPWithRetry monitors the connection and if no data is received after a period of time it will auto-reconnect.
func DialWithErrorHandler(to *net.UDPAddr, onError func()) (*UDPConnWithErrorHandler, error) {
        conn, err := net.DialUDP("udp", nil, to)
        if err != nil {
                return nil, err
        }
        return &UDPConnWithErrorHandler{conn, onError}, nil
}

func (c *UDPConnWithErrorHandler) Write(b []byte) (int, error) {
        c.UDPConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
        n, err := c.UDPConn.Write(b)
        if err != nil {
                defer c.onError()
        }
        return n, err
}

func (c *UDPConnWithErrorHandler) Read(b []byte) (int, error) {
        c.UDPConn.SetReadDeadline(time.Now().Add(5 * time.Second))
        n, err := c.UDPConn.Read(b)
        if err != nil {
                defer c.onError()
        }
        return n, err
}

func (c *UDPConnWithErrorHandler) Close() error {
        return c.UDPConn.Close()
}