package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/balancer"
	"github.com/muxable/rtpmagic/pkg/muxer/encoder"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/sfu/api"
	"github.com/muxable/signal/pkg/signal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	addr := flag.String("addr", "172.26.127.244:50051", "grpc dial address")
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

	enc, err := encoder.NewEncoder()
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

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(audioCodec.RTPCodecCapability, string(audioSource.SSRC()), *cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create audio track")
	}
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(videoCodec.RTPCodecCapability, string(videoSource.SSRC()), *cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create video track")
	}

	go rtpio.CopyRTP(audioTrack, audioSource)
	go rtpio.CopyRTP(videoTrack, videoSource)

	log.Printf("dialing")
	grpcConn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial grpc")
	}

	log.Printf("dialed")

	client := api.NewSFUClient(grpcConn)

	conn, err := client.Signal(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create signal connection")
	}

	udpConn, err := balancer.NewBalancedUDPConn(1 * time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create udp conn")
	}
	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetICEUDPMux(webrtc.NewICEUDPMux(nil, udpConn))

	pcapi := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	pc, err := pcapi.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create peer connection")
	}

	signaller := signal.Negotiate(pc)

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				return
			}
			if err := conn.Send(&api.Request{Operation: &api.Request_Signal{Signal: signal}}); err != nil {
				return
			}
		}
	}()

	go func() {
		for {
			in, err := conn.Recv()
			if err != nil {
				return
			}

			if err := signaller.WriteSignal(in.Signal); err != nil {
				return
			}
		}
	}()

	log.Printf("adding track")
	audioRtpSender, err := pc.AddTrack(audioTrack)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to add audio track")
	}

	videoRtpSender, err := pc.AddTrack(videoTrack)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to add video track")
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// send a clock synchronization packet.
				senderClockPacket, err := packets.NewSenderClockRawPacket(time.Now())
				if err != nil {
					log.Warn().Err(err).Msg("failed to create sender clock packet")
					continue
				}
				if _, err := audioRtpSender.Transport().WriteRTCP([]rtcp.Packet{&senderClockPacket}); err != nil {
					log.Warn().Err(err).Msg("failed to write sender clock packet")
					continue
				}
				if _, err := videoRtpSender.Transport().WriteRTCP([]rtcp.Packet{&senderClockPacket}); err != nil {
					log.Warn().Err(err).Msg("failed to write sender clock packet")
					continue
				}

				// also update the bitrate in this loop because this is a convenient place to do it.
				bitrate, loss := udpConn.GetEstimatedBitrate()
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
			pkts, _, err := videoRtpSender.ReadRTCP()
			if err != nil {
				return
			}
			log.Printf("%v", pkts)
		}
	}()

	go func() {
		for {
			pkts, _, err := audioRtpSender.ReadRTCP()
			if err != nil {
				return
			}
			log.Printf("%v", pkts)
			/*
				if err != nil {
					log.Warn().Err(err).Msg("connection error")
					return
				}
				for _, pkt := range pkts {
					switch pkt := pkt.(type) {
					case *rtcp.PictureLossIndication:
						log.Info().Msg("PLI")
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
					}
				}*/
		}
	}()

	select {}
}
