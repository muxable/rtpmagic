package main

import (
	"flag"
	"io"
	"net"

	"github.com/muxable/rtpmagic/pkg/gstreamer"
	"github.com/muxable/rtpmagic/test/netsim"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

func dial(destination string, useNetsim bool) (io.ReadWriteCloser, error) {
	if useNetsim {
		return netsim.NewNetSimUDPConn(destination, []*netsim.ConnectionState{
			{
				DropRate:      0.10,
				DuplicateRate: 0.10,
			},
			{
				DropRate:      0.10,
				DuplicateRate: 0.10,
			},
			{
				DropRate:      0.10,
				DuplicateRate: 0.10,
			},
		})
	}
	addr, err := net.ResolveUDPAddr("udp", destination)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, addr)
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
	flag.Parse()

	conn, err := dial(*destination, *netsim)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial")
	}

	// create a new gstreamer pipeline.
	var pipelineStr string
	if *test {
		pipelineStr = `
			videotestsrc !
				vp8enc error-resilient=1 deadline=1 cpu-used=5 keyframe-max-dist=10 auto-alt-ref=1 ! rtpvp8pay pt=96 ! appsink name=videosink
			audiotestsrc ! audioresample ! audio/x-raw,channels=1,rate=48000 !
				opusenc inband-fec=true packet-loss-percentage=10 ! rtpopuspay pt=111 ! appsink name=audiosink`
	} else if rtmp != nil {
		// TODO: pipeline string for rtmp.
	}

	audioSSRC := uint32(0)
	videoSSRC := uint32(0)

	pipeline := gstreamer.NewPipeline(pipelineStr)

	pipeline.OnRTPPacket(func(p *rtp.Packet) {
		switch p.PayloadType {
		case 96:
			videoSSRC = p.SSRC
		case 111:
			audioSSRC = p.SSRC
		}
		b, err := p.Marshal()
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal rtp packet")
			return
		}
		if _, err := conn.Write(b); err != nil {
			log.Error().Err(err).Msg("failed to write rtp packet")
			return
		}
	})

	pipeline.Start()

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
				log.Info().Msg("Transport Layer Nack")
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
