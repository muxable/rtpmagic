package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/demuxer"
	"github.com/muxable/rtpmagic/pkg/jitterbuffer"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/muxable/rtpmagic/pkg/receiver"
	"github.com/muxable/rtpmagic/pkg/sender"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
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
			{
				PayloadType: 96,
				RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
				Depacketizer: &codecs.VP8Packet{},
			},
			{
				PayloadType: 111,
				RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000},
				Depacketizer: &codecs.OpusPacket{},
			},
		}),
		Clock:      clock.New(),
		SenderSSRC: rand.Uint32(),
	}
	rtcpReturn := make(chan rtcp.CompoundPacket)

	// receive inbound packets.
	rtpIn, rtcpIn, err := receiver.NewReceiver(ctx, "0.0.0.0:5000", 200*time.Millisecond, rtcpReturn)
	if err != nil {
		panic(err)
	}

	demuxer.NewCNAMEDemuxer(ctx, rtpIn, rtcpIn, func(cnameSource *demuxer.CNAMESource) {
		demuxer.NewSSRCDemuxer(ctx, cnameSource.RTP, cnameSource.RTCP, func(ssrcSource *demuxer.SSRCSource) {
			jb, nack := jitterbuffer.NewCompositeJitterBuffer(ctx, ssrcSource.RTP, []time.Duration{250 * time.Millisecond, 250 * time.Millisecond}, 350*time.Millisecond)
			go func() {
				for n := range nack {
					p := rtcp.TransportLayerNack{
						MediaSSRC:  ssrcSource.SSRC,
						SenderSSRC: ctx.SenderSSRC,
						Nacks:      n,
					}
					log.Printf("sending nack %v", p)
					rtcpReturn <- rtcp.CompoundPacket{&p}
				}
			}()
			demuxer.NewPayloadTypeDemuxer(ctx, jb, func(payloadTypeSource *demuxer.PayloadTypeSource) {
				// match with a codec.
				codec, ok := ctx.Codecs.FindByPayloadType(payloadTypeSource.PayloadType)
				if !ok {
					log.Warn().Uint8("PayloadType", payloadTypeSource.PayloadType).Msg("unknown payload type")
					// we do need to consume all the packets though.
					for range payloadTypeSource.RTP {
					}
					return
				}

				log.Info().Str("CNAME", cnameSource.CNAME).Uint32("SSRC", ssrcSource.SSRC).Uint8("PayloadType", payloadTypeSource.PayloadType).Msg("new inbound stream")

				if err := sender.NewRTPSender(
					"34.72.248.242:50051",
					cnameSource.CNAME,  // broadcast to
					fmt.Sprintf("%s-%d-%d", cnameSource.CNAME, ssrcSource.SSRC, payloadTypeSource.PayloadType), // identify as
					codec,
					payloadTypeSource.RTP); err != nil {
					panic(err)
				}
			})
		})
	})
}
