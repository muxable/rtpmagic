package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/muxable/rtpmagic/pkg/server"
	demuxer "github.com/muxable/rtpmagic/pkg/server/1_demuxer"
	"github.com/muxable/rtpmagic/pkg/server/srt"
	"github.com/muxable/rtptools/pkg/rfc7005"
	"github.com/muxable/rtptools/pkg/x_ssrc"
	sdk "github.com/pion/ion-sdk-go"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
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
	go func() {
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.Handler())
		srv := &http.Server{
			Handler: m,
		}

		metricsLis, err := net.Listen("tcp", ":8012")
		if err != nil {
			return
		}

		err = srv.Serve(metricsLis)
		if err != nil {
			return
		}
	}()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	from := flag.String("from", "0.0.0.0:5000", "The address to receive from")
	to := flag.String("to", "34.150.236.219:50051", "The address to send to")
	useSRTSink := flag.Bool("srt", false, "Use an srt sink")
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

	log.Printf("listening on %s", udpAddr)

	rtpReader, rtcpReader, rtcpWriter := server.NewSSRCManager(ctx, conn, 1500)

	senderSSRC := rand.Uint32()

	if *useSRTSink {
		rtpCh := make(chan *rtp.Packet)
		
		go srt.NewSRTSink(rtpCh)

		x_ssrc.NewDemultiplexer(ctx.Clock.Now, rtpReader, rtcpReader, func(ssrc webrtc.SSRC, rtpIn rtpio.RTPReader, rtcpIn rtpio.RTCPReader) {
			demuxer.NewPayloadTypeDemuxer(ctx.Clock.Now, rtpIn, func(pt webrtc.PayloadType, rtpIn rtpio.RTPReader) {
				// match with a codec.
				codec, ok := ctx.Codecs.FindByPayloadType(pt)
				if !ok {
					log.Warn().Uint32("SSRC", uint32(ssrc)).Uint8("PayloadType", uint8(pt)).Msg("demuxer unknown payload type")
					// we do need to consume all the packets though.
					for {
						p := &rtp.Packet{}
						if _, err := rtpIn.ReadRTP(p); err != nil {
							return
						}
					}
				} else {
					log.Debug().Uint32("SSRC", uint32(ssrc)).Uint8("PayloadType", uint8(pt)).Msg("demuxer found new stream type")
				}
				codecTicker := codec.Ticker()
				defer codecTicker.Stop()
				jb, jbRTP := rfc7005.NewJitterBuffer(codec.ClockRate, 1*time.Second, rtpIn)
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
							if len(missing) == 0 {
								break
							}
							nack := &rtcp.TransportLayerNack{
								SenderSSRC: senderSSRC,
								MediaSSRC:  uint32(ssrc),
								Nacks:      rtcp.NackPairsFromSequenceNumbers(missing),
							}
							if _, err := rtcpWriter.WriteRTCP([]rtcp.Packet{nack}); err != nil {
								log.Error().Err(err).Msg("failed to write NACK")
							}
						case <-done:
							return
						}
					}
				}()

				log.Info().Str("CNAME", "").Uint32("SSRC", uint32(ssrc)).Uint8("PayloadType", uint8(pt)).Msg("new inbound stream")

				for {
					p := &rtp.Packet{}
					if _, err := jbRTP.ReadRTP(p); err != nil {
						log.Error().Err(err).Msg("failed to read RTP")
						break
					}
					rtpCh <- p
				}
			})
		})
	} else {
		connector := sdk.NewConnector(*to)
		rtc := sdk.NewRTC(connector, sdk.DefaultConfig)

		rid := "mugit"

		if err := rtc.Join(rid, sdk.RandomKey(4), sdk.NewJoinConfig().SetNoSubscribe()); err != nil {
			panic(err)
		}

		x_ssrc.NewDemultiplexer(ctx.Clock.Now, rtpReader, rtcpReader, func(ssrc webrtc.SSRC, rtpIn rtpio.RTPReader, rtcpIn rtpio.RTCPReader) {
			demuxer.NewPayloadTypeDemuxer(ctx.Clock.Now, rtpIn, func(pt webrtc.PayloadType, rtpIn rtpio.RTPReader) {
				// match with a codec.
				codec, ok := ctx.Codecs.FindByPayloadType(pt)
				if !ok {
					log.Warn().Uint32("SSRC", uint32(ssrc)).Uint8("PayloadType", uint8(pt)).Msg("demuxer unknown payload type")
					// we do need to consume all the packets though.
					for {
						p := &rtp.Packet{}
						if _, err := rtpIn.ReadRTP(p); err != nil {
							return
						}
					}
				} else {
					log.Debug().Uint32("SSRC", uint32(ssrc)).Uint8("PayloadType", uint8(pt)).Msg("demuxer found new stream type")
				}
				codecTicker := codec.Ticker()
				defer codecTicker.Stop()
				jb, jbRTP := rfc7005.NewJitterBuffer(codec.ClockRate, 1*time.Second, rtpIn)
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
							if len(missing) == 0 {
								break
							}
							nack := &rtcp.TransportLayerNack{
								SenderSSRC: senderSSRC,
								MediaSSRC:  uint32(ssrc),
								Nacks:      rtcp.NackPairsFromSequenceNumbers(missing),
							}
							if _, err := rtcpWriter.WriteRTCP([]rtcp.Packet{nack}); err != nil {
								log.Error().Err(err).Msg("failed to write NACK")
							}
						case <-done:
							return
						}
					}
				}()

				log.Info().Str("CNAME", "").Uint32("SSRC", uint32(ssrc)).Uint8("PayloadType", uint8(pt)).Msg("new inbound stream")

				identity := fmt.Sprintf("%s-%d-%d", "mugit", ssrc, pt)
				if err := NewRTPSender(rtc, identity, codec, jbRTP); err != nil {
					log.Error().Err(err).Msg("sender terminated")
				}
			})
		})
	}
}

func NewRTPSender(rtc *sdk.RTC, tid string, codec *packets.Codec, rtpIn rtpio.RTPReader) error {
	track, err := webrtc.NewTrackLocalStaticRTP(codec.RTPCodecCapability, tid, tid)
	if err != nil {
		return err
	}
	transceivers, err := rtc.Publish(track)
	if err != nil {
		return err
	}
	defer rtc.UnPublish(transceivers...)
	prevSeq := uint16(0)
	for {
		p := &rtp.Packet{}
		if _, err := rtpIn.ReadRTP(p); err != nil {
			return nil
		}
		if p.SequenceNumber != prevSeq + 1 {
			log.Warn().Uint16("PrevSeq", prevSeq).Uint16("CurrSeq", p.SequenceNumber).Msg("missing packet")
		}
		prevSeq = p.SequenceNumber
		if err := track.WriteRTP(p); err != nil {
			log.Warn().Err(err).Msg("failed to write sample")
		}
	}
}
