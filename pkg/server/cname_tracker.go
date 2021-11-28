package server

import (
	"sync"

	"github.com/muxable/rtpio"
	"github.com/pion/rtcp"
)

type cnameTracker struct {
	sync.RWMutex

	cname *string
	src   rtpio.RTCPReader
}

func NewCNAMETracker(rtcpIn rtpio.RTCPReader) *cnameTracker {
	c := &cnameTracker{src: rtcpIn}
	go c.start()
	return c
}

func (c *cnameTracker) start() {
	for {
		pkts := make([]rtcp.Packet, 16)
		n, err := c.src.ReadRTCP(pkts)
		if err != nil {
			return
		}
		for _, pkt := range pkts[:n] {
			switch p := pkt.(type) {
			case *rtcp.SourceDescription:
				for _, chunk := range p.Chunks {
					for _, item := range chunk.Items {
						if item.Type == rtcp.SDESCNAME {
							c.Lock()
							c.cname = &item.Text
							c.Unlock()
						}
					}
				}
			}
		}
	}
}

// Get retrieves the active CNAME.
func (c *cnameTracker) Get() *string {
	c.RLock()
	defer c.RUnlock()
	return c.cname
}
