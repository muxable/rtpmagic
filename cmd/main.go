package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/muxable/rtpmagic/api"
	"github.com/muxable/rtpmagic/pkg/muxer/encoder"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/signal/pkg/signal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/cc"
	"github.com/pion/interceptor/pkg/gcc"
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
	videoSrc := flag.String("video-src", "v4l2src device=/dev/video0", "GStreamer video src")
	// netsim := flag.Bool("netsim", false, "enable network simulation")
	dest := flag.String("dest", "100.105.100.81:50051", "rtp sink destination")
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

	conn, err := grpc.Dial(*dest, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := api.NewSFUClient(conn)

	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
			RTPCodecCapability: videoCodec.RTPCodecCapability,
			PayloadType:        96,
		}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
			RTPCodecCapability: audioCodec.RTPCodecCapability,
			PayloadType:        97,
		}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	i := &interceptor.Registry{}

	congestionController, err := cc.NewInterceptor(func() (cc.BandwidthEstimator, error) {
		return gcc.NewSendSideBWE(gcc.SendSideBWEInitialBitrate(300_000))
	})
	if err != nil {
		panic(err)
	}

	congestionController.OnNewPeerConnection(func(id string, estimator cc.BandwidthEstimator) {
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			for range ticker.C {
				bitrate := estimator.GetTargetBitrate()
				if bitrate > 64000 {
					bitrate -= 64000 // subtract off audio bitrate
				}
				if bitrate < 100000 {
					bitrate = 100000
				}
				log.Printf("got bitrate %v", bitrate)
				videoSource.SetBitrate(uint32(bitrate))
				// audioSource.SetPacketLossPercentage(uint32(loss * 100))
			}
		}()
	})

	i.Add(congestionController)
	if err = webrtc.ConfigureTWCCHeaderExtensionSender(m, i); err != nil {
		panic(err)
	}

	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		panic(err)
	}

	pc, err := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		panic(err)
	}

	stream, err := client.Signal(context.Background())
	if err != nil {
		panic(err)
	}

	signaller := signal.NewSignaller(pc)

	go func() {
		for {
			pb, err := signaller.ReadSignal()
			if err != nil {
				panic(err)
			}
			log.Printf("%v", pb)
			if err := stream.Send(pb); err != nil {
				panic(err)
			}
		}
	}()

	pc.OnNegotiationNeeded(signaller.Renegotiate)

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(audioCodec.RTPCodecCapability, uuid.NewString(), *cname)
	if err != nil {
		panic(err)
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(videoCodec.RTPCodecCapability, uuid.NewString(), *cname)
	if err != nil {
		panic(err)
	}

	audioSender, err := pc.AddTrack(audioTrack)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			_, _, err := audioSender.ReadRTCP()
			if err != nil {
				panic(err)
			}
		}
	}()

	videoSender, err := pc.AddTrack(videoTrack)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			_, _, err := videoSender.ReadRTCP()
			if err != nil {
				panic(err)
			}
		}
	}()

	go rtpio.CopyRTP(audioTrack, audioSource)
	go rtpio.CopyRTP(videoTrack, videoSource)

	for {
		pb, err := stream.Recv()
		if err != nil {
			panic(err)
		}
		log.Printf("%v", pb)
		if err := signaller.WriteSignal(pb); err != nil {
			panic(err)
		}
	}

	// go func() {
	// 	ticker := time.NewTicker(100 * time.Millisecond)
	// 	defer ticker.Stop()
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			if err := conn.WriteRTCP([]rtcp.Packet{SourceDescription(*cname, audioSource.SSRC()), SourceDescription(*cname, videoSource.SSRC())}); err != nil {
	// 				log.Warn().Err(err).Msg("failed to write rtcp packet")
	// 			}

	// 			// also update the bitrate in this loop because this is a convenient place to do it.
	// 			bitrate, loss := conn.GetEstimatedBitrate()
	// 			if bitrate > 64000 {
	// 				bitrate -= 64000 // subtract off audio bitrate
	// 			}
	// 			if bitrate < 100000 {
	// 				bitrate = 100000
	// 			}
	// 			videoSource.SetBitrate(bitrate)
	// 			audioSource.SetPacketLossPercentage(uint32(loss * 100))
	// 		}
	// 	}
	// }()

	// go func() {
	// 	for {
	// 		pkts, err := conn.ReadRTCP()
	// 		if err != nil {
	// 			log.Warn().Err(err).Msg("connection error")
	// 			return
	// 		}
	// 		for _, pkt := range pkts {
	// 			switch pkt := pkt.(type) {
	// 			case *rtcp.PictureLossIndication:
	// 				log.Info().Msg("PLI")
	// 			case *rtcp.ReceiverReport:
	// 				log.Info().Msg("Receiver Report")
	// 			case *rtcp.Goodbye:
	// 				log.Info().Msg("Goodbye")
	// 			case *rtcp.TransportLayerNack:
	// 				for _, nack := range pkt.Nacks {
	// 					for _, id := range nack.PacketList() {
	// 						switch webrtc.SSRC(pkt.MediaSSRC) {
	// 						case videoSource.SSRC():
	// 							if _, q := videoSendBuffer.Get(id); q != nil {
	// 								log.Warn().Uint16("Seq", id).Msg("responding to video nack")
	// 								if err := conn.WriteRTP(q); err != nil {
	// 									log.Error().Err(err).Msg("failed to write rtp")
	// 								}
	// 							} else {
	// 								log.Warn().Uint16("Seq", id).Msg("nack referring to missing packet")
	// 							}
	// 						case audioSource.SSRC():
	// 							if _, q := audioSendBuffer.Get(id); q != nil {
	// 								if err := conn.WriteRTP(q); err != nil {
	// 									log.Error().Err(err).Msg("failed to write rtp")
	// 								}
	// 							} else {
	// 								log.Warn().Uint16("Seq", id).Msg("nack referring to missing packet")
	// 							}
	// 						default:
	// 							log.Error().Uint32("SSRC", pkt.MediaSSRC).Msg("nack referring to unknown ssrc")
	// 						}
	// 					}
	// 				}
	// 			case *rtcp.TransportLayerCC:
	// 				log.Info().Msg("Transport Layer CC")
	// 			default:
	// 				// log.Info().Msgf("unknown rtcp packet: %v", pkt)
	// 			}
	// 		}
	// 	}
	// }()
}
