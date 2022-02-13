package main

import (
	"flag"
	"os"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer"
	"github.com/muxable/rtpmagic/pkg/muxer/encoder"
	"github.com/muxable/rtpmagic/pkg/muxer/nack"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/pion/rtcp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

func SourceDescription(cname string, ssrc webrtc.SSRC) *rtcp.SourceDescription {
	return &rtcp.SourceDescription{
		Chunks: []rtcp.SourceDescriptionChunk{{
			Source: uint32(ssrc),
			Items:  []rtcp.SourceDescriptionItem{{Type: rtcp.SDESCNAME, Text: cname}},
		}},
	}
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cname := flag.String("cname", "mugit", "join session name")
	audioSrc := flag.String("audio-src", "alsasrc device=hw:2", "GStreamer audio src")
	videoSrc := flag.String("video-src", "v4l2src", "GStreamer video src")
	netsim := flag.Bool("netsim", false, "enable network simulation")
	dest := flag.String("dest", "34.85.161.200:5000", "rtp sink destination")
	flag.Parse()

	codecs := packets.DefaultCodecSet()
	videoCodec, ok := codecs.FindByMimeType(webrtc.MimeTypeH265)
	if !ok {
		log.Fatal().Msg("failed to find video codec")
	}
	audioCodec, ok := codecs.FindByMimeType(webrtc.MimeTypeOpus)
	if !ok {
		log.Fatal().Msg("failed to find audio codec")
	}

	conn, err := muxer.Dial(*dest, *netsim)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial rtp sink")
	}

	enc, err := encoder.NewEncoder(*cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create encoder")
	}

	audioSource, err := enc.AddSource(*audioSrc, audioCodec)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create audio encoder")
	}

	videoSource, err := enc.AddSource(*videoSrc, videoCodec)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create video encoder")
	}

	audioSendBuffer := nack.NewSendBuffer(14)
	videoSendBuffer := nack.NewSendBuffer(14)

	if err := conn.WriteRTCP([]rtcp.Packet{SourceDescription(*cname, audioSource.SSRC()), SourceDescription(*cname, videoSource.SSRC())}); err != nil {
		log.Fatal().Err(err).Msg("failed to write rtcp packet")
	}

	go rtpio.CopyRTP(conn, audioSource)
	go rtpio.CopyRTP(conn, videoSource)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteRTCP([]rtcp.Packet{SourceDescription(*cname, audioSource.SSRC()), SourceDescription(*cname, videoSource.SSRC())}); err != nil {
					log.Fatal().Err(err).Msg("failed to write rtcp packet")
				}

				// also update the bitrate in this loop because this is a convenient place to do it.
				bitrate, loss := conn.GetEstimatedBitrate()
				if bitrate > 64000 {
					bitrate -= 64000 // subtract off audio bitrate
				}
				if bitrate < 100000 {
					bitrate = 100000
				}
				videoSource.SetBitrate(bitrate)
				audioSource.SetPacketLossPercentage(uint32(loss * 100))
			}
		}
	}()

	go func() {
		for {
			pkts, err := conn.ReadRTCP()
			if err != nil {
				log.Warn().Err(err).Msg("connection error")
				return
			}
			for _, pkt := range pkts {
				switch pkt := pkt.(type) {
				case *rtcp.PictureLossIndication:
					log.Info().Msg("PLI")
				case *rtcp.ReceiverReport:
					log.Info().Msg("Receiver Report")
				case *rtcp.Goodbye:
					log.Info().Msg("Goodbye")
				case *rtcp.TransportLayerNack:
					for _, nack := range pkt.Nacks {
						for _, id := range nack.PacketList() {
							switch webrtc.SSRC(pkt.MediaSSRC) {
							case videoSource.SSRC():
								if _, q := videoSendBuffer.Get(id); q != nil {
									log.Warn().Uint16("Seq", id).Msg("responding to video nack")
									if err := conn.WriteRTP(q); err != nil {
										log.Error().Err(err).Msg("failed to write rtp")
									}
								} else {
									log.Warn().Uint16("Seq", id).Msg("nack referring to missing packet")
								}
							case audioSource.SSRC():
								if _, q := audioSendBuffer.Get(id); q != nil {
									if err := conn.WriteRTP(q); err != nil {
										log.Error().Err(err).Msg("failed to write rtp")
									}
								} else {
									log.Warn().Uint16("Seq", id).Msg("nack referring to missing packet")
								}
							default:
								log.Error().Uint32("SSRC", pkt.MediaSSRC).Msg("nack referring to unknown ssrc")
							}
						}
					}
				case *rtcp.TransportLayerCC:
					log.Info().Msg("Transport Layer CC")
				default:
					// log.Info().Msgf("unknown rtcp packet: %v", pkt)
				}
			}
		}
	}()

	select {}
}
