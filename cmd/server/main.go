package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/muxable/rtpmagic/pkg/server"
	demuxer "github.com/muxable/rtpmagic/pkg/server/1_demuxer"
	sender "github.com/muxable/rtpmagic/pkg/server/3_sender"
	"github.com/muxable/rtptools/pkg/report"
	"github.com/muxable/rtptools/pkg/rfc7005"
	"github.com/muxable/rtptools/pkg/x_ssrc"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
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
	from := flag.String("from", "0.0.0.0:5000", "The address to receive from")
	to := flag.String("to", "34.72.248.242:50051", "The address to send to")
	flag.Parse()

	ctx := pipeline.Context{
		Codecs:     packets.DefaultCodecSet(),
		Clock:      clock.New(),
		SenderSSRC: rand.Uint32(),
	}

	// receive inbound packets.
	udpAddr, err := net.ResolveUDPAddr("udp", *from)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to resolve UDP address")
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen on UDP")
	}

	// md := feature.NewMediaDescriptionReceiver(ctx.Codecs.RTPCodecParameters(), []string{"nada", "nack", "flex-fec"})

	// rr, err := report.NewReceiverInterceptor(md)
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("failed to create receiver interceptor")
	// }

	rtpReader, rtcpReader, rtcpWriter := server.NewSSRCManager(ctx, conn, 1500)

	senderSSRC := rand.Uint32()
 
	x_ssrc.NewDemultiplexer(ctx.Clock.Now, rtpReader, rtcpReader, func(ssrc webrtc.SSRC, rtpIn rtpio.RTPReader, rtcpIn rtpio.RTCPReader) {
		demuxer.NewPayloadTypeDemuxer(ctx.Clock.Now, rtpIn, func(pt webrtc.PayloadType, rtpIn rtpio.RTPReader) {
			// match with a codec.
			codec, ok := ctx.Codecs.FindByPayloadType(pt)
			if !ok {
				log.Warn().Uint8("PayloadType", uint8(pt)).Msg("demuxer unknown payload type")
				// we do need to consume all the packets though.
				for {
					p := &rtp.Packet{}
					if _, err := rtpIn.ReadRTP(p); err != nil {
						return
					}
				}
			}
			rrRTPReader, rrRTPWriter := rtpio.RTPPipe()
			rtpIn = rtpio.TeeRTPReader(rtpIn, rrRTPWriter)
			rr := report.NewReceiverStream(ctx.Clock.Now, uint32(ssrc), codec.ClockRate, rrRTPReader, rtcpIn)

			codecTicker := codec.Ticker()
			defer codecTicker.Stop()
			jb, jbRTP := rfc7005.NewJitterBuffer(codec.ClockRate, 2 * time.Second, rtpIn)
			// write nacks periodically back to the sender
			nackTicker := time.NewTicker(150 * time.Millisecond)
			defer nackTicker.Stop()
			done := make(chan bool, 1)
			defer func() { done <- true }()
			go func() {
				for {
					select {
					case <-nackTicker.C:
						missing := jb.GetMissingSequenceNumbers(uint64(codec.ClockRate / 10))
						nack := &rtcp.TransportLayerNack{
							SenderSSRC: senderSSRC,
							MediaSSRC:  uint32(ssrc),
							Nacks:      rtcp.NackPairsFromSequenceNumbers(missing),
						}
						if _, err := rtcpWriter.WriteRTCP([]rtcp.Packet{
							rr.GenerateReport(),
							nack,
						}); err != nil {
							log.Error().Err(err).Msg("failed to write NACK")
						}
					case <-done:
						return
					}
				}
			}()

			log.Info().Str("CNAME", "").Uint32("SSRC", uint32(ssrc)).Uint8("PayloadType", uint8(pt)).Msg("new inbound stream")

			identity := fmt.Sprintf("%s-%d-%d", "mugit", ssrc, pt)
			if err := sender.NewRTPSender(*to, "mugit", identity, codec, jbRTP); err != nil {
				log.Error().Err(err).Msg("sender terminated")
			}
		})
	})
}
