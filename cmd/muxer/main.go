package main

import (
	"flag"
	"fmt"
	"io"
	"net"

	"github.com/muxable/rtpmagic/pkg/gstreamer"
	"github.com/muxable/rtpmagic/pkg/nack"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/test/netsim"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

func dial(destination string, useNetsim bool) (io.ReadWriteCloser, error) {
	if useNetsim {
		return netsim.NewNetSimUDPConn(destination, []*netsim.ConnectionState{
			{
				DropRate:      0.30,
			},
		})
	}
	addr, err := net.ResolveUDPAddr("udp", destination)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, addr)
}

func writeRTP(conn io.ReadWriter, p *rtp.Packet) error {
	b, err := p.Marshal()
	if err != nil {
		return err
	}
	if _, err := conn.Write(b); err != nil {
		return err
	}
	return nil
}

// the general pipeline is
// audio/video src as string -> raw
// encode audio and video
// broadcast rtp
func main() {
	rtmp := flag.String("rtmp", "", "rtmp url")
	test := flag.Bool("test", false, "use test src")
	netsim := flag.Bool("netsim", false, "use netsim connection")
	destination := flag.String("dest", "", "destination")
	audioMimeType := flag.String("audio", webrtc.MimeTypeOpus, "audio mime type")
	videoMimeType := flag.String("video", webrtc.MimeTypeVP8, "video mime type")
	flag.Parse()

	audioCodec, ok := packets.DefaultCodecSet().FindByMimeType(*audioMimeType)
	if !ok {
		log.Fatal().Msg("no audio codec")
	}
	videoCodec, ok := packets.DefaultCodecSet().FindByMimeType(*videoMimeType)
	if !ok {
		log.Fatal().Msg("no video codec")
	}

	conn, err := dial(*destination, *netsim)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial")
	}

	videoSendBuffer, err := nack.NewSendBuffer(1 << 14)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create send buffer")
	}
	audioSendBuffer, err := nack.NewSendBuffer(1 << 12)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create send buffer")
	}

	// create a new gstreamer pipeline.
	var pipelineStr string
	if *test {
		pipelineStr = fmt.Sprintf(`
			videotestsrc ! %s ! appsink name=videosink
			audiotestsrc ! %s ! appsink name=audiosink`,
			videoCodec.GStreamerPipeline,
			audioCodec.GStreamerPipeline)
	} else if rtmp != nil {
		// TODO: pipeline string for rtmp.
	}

	audioSSRC := uint32(0)
	videoSSRC := uint32(0)

	pipeline := gstreamer.NewPipeline(pipelineStr)

	pipeline.OnRTPPacket(func(p *rtp.Packet) {
		switch p.PayloadType {
		case videoCodec.PayloadType:
			videoSSRC = p.SSRC
			videoSendBuffer.Add(p)
			log.Printf("sending ts %d", p.Timestamp)
		case audioCodec.PayloadType:
			audioSSRC = p.SSRC
			audioSendBuffer.Add(p)
		}
		if err := writeRTP(conn, p); err != nil {
			log.Error().Err(err).Msg("failed to write rtp")
		}
	})

	pipeline.Start()

	// ctx, cancel := context.WithCancel(context.Background())
	// videoSenderStream := reports.NewSenderStream(videoCodec.ClockRate)
	// audioSenderStream := reports.NewSenderStream(audioCodec.ClockRate)
	// go func() {
	// 	// send forward SDES and SR packets over RTCP
	// 	ticker := time.NewTicker(2 * time.Second)
	// 	defer ticker.Stop()
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 		case <-ticker.C:
	// 			now := time.Now()
	// 			// build feedback for video ssrc.
	// 			videoPacket := videoSenderStream.BuildFeedbackPacket(now, videoSSRC)
				



	// 		}
	// }()

	for {
		buf := make([]byte, 1500)
		n, err := conn.Read(buf)
		if err != nil {
			log.Warn().Err(err).Msg("connection error")
			return
		}
		// assume these are all rtcp packets.
		cp, err := rtcp.Unmarshal(buf[:n])
		if err != nil {
			log.Warn().Err(err).Msg("rtcp unmarshal error")
			continue
		}
		for _, p := range cp {
			// we might get packets for unrelated SSRCs, so discard this packet if it's not relevant to us.
			if !containsSSRC(p.DestinationSSRC(), videoSSRC) && !containsSSRC(p.DestinationSSRC(), audioSSRC) {
				log.Warn().Msg("discarding packet for unrelated SSRC")
				continue
			}
			switch p := p.(type) {
			case *rtcp.PictureLossIndication:
				log.Info().Msg("PLI")
			case *rtcp.ReceiverReport:
				log.Info().Msg("Receiver Report")
			case *rtcp.Goodbye:
				log.Info().Msg("Goodbye")
			case *rtcp.TransportLayerNack:
				for _, nack := range p.Nacks {
					for _, id := range nack.PacketList() {
						switch p.MediaSSRC {
						case videoSSRC:
							q := videoSendBuffer.Get(id)
							if q != nil {
								if err := writeRTP(conn, q); err != nil {
									log.Error().Err(err).Msg("failed to write rtp")
								}
							}
						case audioSSRC:
							q := audioSendBuffer.Get(id)
							if q != nil {
								if err := writeRTP(conn, q); err != nil {
									log.Error().Err(err).Msg("failed to write rtp")
								}
							}
						}
					}
				}
			case *rtcp.TransportLayerCC:
				log.Info().Msg("Transport Layer CC")
			default:
				log.Warn().Interface("Packet", p).Msg("unknown rtcp packet")
			}
		}
	}
}

func containsSSRC(ssrcs []uint32, ssrc uint32) bool {
	for _, s := range ssrcs {
		if s == ssrc {
			return true
		}
	}
	return false
}
