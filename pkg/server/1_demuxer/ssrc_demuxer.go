package demuxer

import (
	"time"

	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type SSRCDemuxer struct {
	ctx      pipeline.Context
	rtpIn    chan *rtp.Packet
	rtcpIn   chan *rtcp.Packet
	callback func(*SSRCSource)

	bySSRC map[uint32]*SSRCSource
}

type SSRCSource struct {
	SSRC uint32
	RTP  chan *rtp.Packet
	RTCP chan *rtcp.Packet

	lastPacket time.Time
}

// NewSSRCDemuxer creates a new SSRCDemuxer
func NewSSRCDemuxer(ctx pipeline.Context, rtpIn chan *rtp.Packet, rtcpIn chan *rtcp.Packet, callback func(*SSRCSource)) {
	d := &SSRCDemuxer{
		ctx:      ctx,
		rtpIn:    rtpIn,
		rtcpIn:   rtcpIn,
		callback: callback,
		bySSRC:   make(map[uint32]*SSRCSource),
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case p, ok := <-d.rtpIn:
			if !ok {
				return
			}
			d.handleRTP(p)
		case p, ok := <-d.rtcpIn:
			if !ok {
				return
			}
			d.handleRTCP(p)
		case <-ticker.C:
			d.cleanup()
		}
	}
}

// handleRTP checks if the RTP's SSRC is registered and if so, forwards it on to that source
func (d *SSRCDemuxer) handleRTP(p *rtp.Packet) {
	s, ok := d.bySSRC[p.SSRC]
	if !ok {
		// create a new SSRC source.
		s = &SSRCSource{
			SSRC: p.SSRC,
			RTP:  make(chan *rtp.Packet),
			RTCP: make(chan *rtcp.Packet),
		}
		d.bySSRC[p.SSRC] = s
		go d.callback(s)
	}
	s.lastPacket = d.ctx.Clock.Now()
	s.RTP <- p
}

// handleRTCP registers a given SSRC to a SSRCSource.
func (d *SSRCDemuxer) handleRTCP(p *rtcp.Packet) {
	for _, ssrc := range (*p).DestinationSSRC() {
		s, ok := d.bySSRC[ssrc]
		if !ok {
			// create a new SSRC source.
			s = &SSRCSource{
				SSRC: ssrc,
				RTP:  make(chan *rtp.Packet),
				RTCP: make(chan *rtcp.Packet),
			}
			d.bySSRC[ssrc] = s
			go d.callback(s)
		}
		s.lastPacket = d.ctx.Clock.Now()
		s.RTCP <- p
	}
}

// cleanup removes any ssrc sources that haven't received a packet in the last 30 seconds
func (d *SSRCDemuxer) cleanup() {
	now := d.ctx.Clock.Now()
	for ssrc, s := range d.bySSRC {
		if now.Sub(s.lastPacket) > 30*time.Second {
			// log the removal
			log.Info().Uint32("SSRC", ssrc).Msg("removing ssrc due to timeout")
			delete(d.bySSRC, ssrc)
			// close the output channels
			close(s.RTP)
			close(s.RTCP)
		}
	}
}
