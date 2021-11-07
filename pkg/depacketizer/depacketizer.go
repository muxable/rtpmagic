package depacketizer

import (
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rs/zerolog/log"
)

type Depacketizer struct {
	ctx pipeline.Context

	in  chan rtp.Packet
	out chan media.Sample
}

// NewDepacketizer creates a new depacketizer.
func NewDepacketizer(ctx pipeline.Context, rtpIn chan rtp.Packet) chan media.Sample {
	d := &Depacketizer{
		ctx: ctx,
		in:  rtpIn,
		out: make(chan media.Sample),
	}
	go d.start()
	return d.out
}

var vp8 = &codecs.VP8Packet{}

// start starts the depacketizer.
func (d *Depacketizer) start() {
	defer close(d.out)
	for p := range d.in {
		codec, ok := d.ctx.Codecs.FindByPayloadType(p.PayloadType)
		if !ok {
			log.Warn().Uint8("PayloadType", p.PayloadType).Msg("Unknown payload type")
			continue
		}
		switch codec.MimeType {
		case webrtc.MimeTypeVP8:
			data, err := vp8.Unmarshal(p.Payload)
			if err != nil {
				log.Error().Err(err).Msg("Unmarshal VP8 packet")
				continue
			}
			d.out <- media.Sample{
				Data: data,
			}
		case webrtc.MimeTypeOpus:

		}
	}
}
