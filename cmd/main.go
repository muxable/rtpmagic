package main

import (
	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/demuxer"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/muxable/rtpmagic/pkg/receiver"
	"github.com/muxable/rtpmagic/pkg/sender"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

// The overall pipeline follows the following architecture:
// - receiver
// - cname demuxer
// - normalizer
// - jitter buffer + nack emitter
// - pt demuxer
// - depacketizer
// - transcoder
// - pt muxer (implicit)
// - sender
func main() {
	ctx := pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{PayloadType: 96, MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
			{PayloadType: 111, MimeType: webrtc.MimeTypeOpus, ClockRate: 48000},
		}),
		Clock: clock.New(),
	}
	rtcpReturn := make(chan *rtcp.Packet)

	log.Info().Msg("listening")

	// receive inbound packets.
	r0, r1, err := receiver.NewReceiver(ctx, "0.0.0.0:5000", rtcpReturn)
	if err != nil {
		panic(err)
	}

	for source := range demuxer.NewCNAMEDemuxer(ctx, r0, r1) {
		vp, ap := demuxer.NewPayloadTypeDemuxer(ctx, source.RTP)
		if err := sender.NewRTPSender(
			"34.72.248.242:50051", source.CNAME,
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeVP8,
				ClockRate: 90000,
			},
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeOpus,
				ClockRate: 48000,
			},
			vp, ap); err != nil {
			panic(err)
		}
	}
	// demux the packets by cname.
	// for source := range demuxer.NewCNAMEDemuxer(ctx, r0, r1) {
		// normalize the packets by SSRC.
		// n := normalizer.NewNormalizer(ctx, source.RTP)

		// // jitter buffer + nack emitter.
		// jb, nack := jitterbuffer.NewCompositeJitterBuffer(ctx, n, []time.Duration{2 * time.Second})

		// go func() {
		// 	for p := range nack {
		// 		log.Info().Msgf("fake nack send %v", p)
		// 	}
		// }()

		// grab the video and audio streams
		// vp, ap := demuxer.NewPayloadTypeDemuxer(ctx, source.RTP)

		// depacketize the packets.
		// v := depacketizer.NewDepacketizer(ctx, vp)
		// a := depacketizer.NewDepacketizer(ctx, ap)

		// transcode the packets.
		// TODO: transcoding.

		// send the packets.
		// if err := sender.NewRTPSender(
		// 	"34.138.20.36:50051", source.CNAME,
		// 	webrtc.RTPCodecCapability{
		// 		MimeType:  webrtc.MimeTypeVP8,
		// 		ClockRate: 90000,
		// 	},
		// 	webrtc.RTPCodecCapability{
		// 		MimeType:  webrtc.MimeTypeOpus,
		// 		ClockRate: 48000,
		// 	},
		// 	vp, ap); err != nil {
		// 	panic(err)
		// }
	// }
}
