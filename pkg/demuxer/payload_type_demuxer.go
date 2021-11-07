package demuxer

import (
	"strings"

	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type PayloadTypeDemuxer struct {
	ctx   pipeline.Context
	rtpIn chan *rtp.Packet

	videoRtpOut chan *rtp.Packet
	audioRtpOut chan *rtp.Packet
}

// NewPayloadTypeDemuxer creates a new PayloadTypeDemuxer
func NewPayloadTypeDemuxer(ctx pipeline.Context, rtpIn chan *rtp.Packet) (chan *rtp.Packet, chan *rtp.Packet) {
	videoRtpOut := make(chan *rtp.Packet)
	audioRtpOut := make(chan *rtp.Packet)

	d := &PayloadTypeDemuxer{
		ctx:         ctx,
		rtpIn:       rtpIn,
		videoRtpOut: videoRtpOut,
		audioRtpOut: audioRtpOut,
	}
	go d.start()

	return videoRtpOut, audioRtpOut
}

// start starts processing the input RTP streams.
func (d *PayloadTypeDemuxer) start() {
	defer close(d.videoRtpOut)
	defer close(d.audioRtpOut)

	for pkt := range d.rtpIn {
		codec, ok := d.ctx.Codecs.FindByPayloadType(pkt.PayloadType)
		if !ok {
			log.Warn().Uint8("PayloadType", pkt.PayloadType).Msg("unknown payload type")
			continue
		}

		// TODO: assert that the video and audio mime types are consistent, ie no changing payload types.
		mt := codec.MimeType
		if strings.HasPrefix(mt, "video") {
			d.videoRtpOut <- pkt
		} else if strings.HasPrefix(mt, "audio") {
			d.audioRtpOut <- pkt
		} else {
			log.Warn().Str("MimeType", mt).Msg("unknown mime type")
		}
	}
}
