package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	receiver "github.com/muxable/rtpmagic/pkg/server/0_receiver"
	demuxer "github.com/muxable/rtpmagic/pkg/server/1_demuxer"
	jitterbuffer "github.com/muxable/rtpmagic/pkg/server/2_jitterbuffer"
	sender "github.com/muxable/rtpmagic/pkg/server/3_sender"
	"github.com/pion/rtcp"
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
	from := flag.String("from", "0.0.0.0:5000", "The address to receive from")
	to := flag.String("to", "34.72.248.242:50051", "The address to send to")
	flag.Parse()

	ctx := pipeline.Context{
		Codecs:     packets.DefaultCodecSet(),
		Clock:      clock.New(),
		SenderSSRC: rand.Uint32(),
	}

	// receive inbound packets.
	rtpIn, rtcpIn, err := receiver.NewReceiver(ctx, *from, 200*time.Millisecond)
	if err != nil {
		panic(err)
	}

	demuxer.NewCNAMEDemuxer(ctx, rtpIn, rtcpIn, func(cnameSource *demuxer.CNAMESource) {
		demuxer.NewSSRCDemuxer(ctx, cnameSource.RTP, cnameSource.RTCP, func(ssrcSource *demuxer.SSRCSource) {
			jb, nack := jitterbuffer.NewCompositeJitterBuffer(ctx, ssrcSource.RTP, []time.Duration{200 * time.Millisecond, 200 * time.Millisecond, 200 * time.Millisecond, 200 * time.Millisecond}, 350*time.Millisecond)
			go func() {
				for n := range nack {
					p := rtcp.TransportLayerNack{
						MediaSSRC:  ssrcSource.SSRC,
						SenderSSRC: ctx.SenderSSRC,
						Nacks:      n,
					}
					log.Printf("sending nack %v", p)
				}
			}()
			demuxer.NewPayloadTypeDemuxer(ctx, jb, func(payloadTypeSource *demuxer.PayloadTypeSource) {
				// match with a codec.
				codec, ok := ctx.Codecs.FindByPayloadType(payloadTypeSource.PayloadType)
				if !ok {
					log.Warn().Uint8("PayloadType", payloadTypeSource.PayloadType).Msg("demuxer unknown payload type")
					// we do need to consume all the packets though.
					for range payloadTypeSource.RTP {
					}
					return
				}

				log.Info().Str("CNAME", cnameSource.CNAME).Uint32("SSRC", ssrcSource.SSRC).Uint8("PayloadType", payloadTypeSource.PayloadType).Msg("new inbound stream")

				if err := sender.NewRTPSender(
					*to,
					cnameSource.CNAME, // broadcast to
					fmt.Sprintf("%s-%d-%d", cnameSource.CNAME, ssrcSource.SSRC, payloadTypeSource.PayloadType), // identify as
					codec,
					payloadTypeSource.RTP); err != nil {
					panic(err)
				}
			})
		})
	})
}
