package demuxer

import (
	"time"

	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type PayloadTypeSource struct {
	PayloadType uint8
	RTP         chan *rtp.Packet

	lastPacket time.Time
}

type PayloadTypeDemuxer struct {
	ctx           pipeline.Context
	rtpIn         chan *rtp.Packet
	byPayloadType map[uint8]*PayloadTypeSource
	callback      func(*PayloadTypeSource)
}

// NewPayloadTypeDemuxer creates a new PayloadTypeDemuxer
func NewPayloadTypeDemuxer(ctx pipeline.Context, rtpIn chan *rtp.Packet, callback func(*PayloadTypeSource)) {

	d := &PayloadTypeDemuxer{
		ctx:           ctx,
		rtpIn:         rtpIn,
		byPayloadType: make(map[uint8]*PayloadTypeSource),
		callback:      callback,
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case p, ok := <-d.rtpIn:
			if !ok {
				return
			}

			s, ok := d.byPayloadType[p.PayloadType]
			if !ok {
				s = &PayloadTypeSource{
					PayloadType: p.PayloadType,
					RTP:         make(chan *rtp.Packet),
				}
				d.byPayloadType[p.PayloadType] = s
				go d.callback(s)
			}
			s.lastPacket = d.ctx.Clock.Now()
			s.RTP <- p
		case <-ticker.C:
			d.cleanup()
		}
	}
}

// cleanup removes any payload types that have been inactive for a while.
func (d *PayloadTypeDemuxer) cleanup() {
	now := d.ctx.Clock.Now()
	for pt, s := range d.byPayloadType {
		if now.Sub(s.lastPacket) > 30*time.Second {
			// log the removal
			log.Info().Uint8("PayloadType", pt).Msg("removing pt due to timeout")
			delete(d.byPayloadType, pt)
			// close the output channels
			close(s.RTP)
		}
	}
}
