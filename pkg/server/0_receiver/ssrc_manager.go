package receiver

import (
	"io"
	"net"
	"sync"

	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type SSRCManager struct {
	sync.RWMutex
	
	ctx pipeline.Context
	conn *net.UDPConn
	sources map[uint32]*net.UDPAddr
}

// NewSSRCManager wraps a net.UDPConn and provides a way to track the SSRCs of the sender.
func NewSSRCManager(ctx pipeline.Context, conn *net.UDPConn) io.ReadWriter {
	return &SSRCManager{
		ctx: ctx,
		conn: conn,
		sources: make(map[uint32]*net.UDPAddr),
	}
}

// Read reads from the connection.
func (m *SSRCManager) Read(buf []byte) (int, error) {
	n, sender, err := m.conn.ReadFromUDP(buf)
	if err != nil {
		return 0, err
	}
	p := &rtp.Header{}
	if _, err := p.Unmarshal(buf[:n]); err != nil {
		log.Warn().Msgf("failed to parse rtp packet: %v", err)
		return 0, err
	}
	if _, ok := m.ctx.Codecs.FindByPayloadType(p.PayloadType); ok {
		m.Lock()
		m.sources[p.SSRC] = sender
		m.Unlock()
	}
	return n, nil
}

// Write writes to the connection sending to only senders that have sent to that ssrc.
func (m *SSRCManager) Write(buf []byte) (int, error) {
	p := &rtp.Header{}
	if _, err := p.Unmarshal(buf); err != nil {
		log.Warn().Msgf("failed to parse rtp packet: %v", err)
		return 0, err
	}
	if _, ok := m.ctx.Codecs.FindByPayloadType(p.PayloadType); !ok {
		// this is an rtcp packet.
		p2, err := rtcp.Unmarshal(buf)
		if err != nil {
			log.Warn().Msgf("failed to parse rtcp packet: %v", err)
			return 0, err
		}
		m.RLock()
		for _, p3 := range p2 {
			for _, ssrcs := range p3.DestinationSSRC() {
				// forward this packet to that ssrc's source.
				if addr, ok := m.sources[ssrcs]; ok {
					if _, err := m.conn.WriteToUDP(buf, addr); err != nil {
						log.Error().Err(err).Msg("failed to send rtcp packet")
					}
				}
			}
		}
		m.RUnlock()
	}
	return len(buf), nil
}
