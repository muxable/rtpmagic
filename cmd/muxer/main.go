package main

import (
	"flag"
	"net"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

// the general pipeline is
// audio/video src as string -> raw
// encode audio and video
// broadcast rtp
func main() {
	rtmp := flag.String("rtmp", "", "rtmp url")
	test := flag.Bool("test", false, "use test src")
	destination := flag.String("dest", "", "destination")
	flag.Parse()

	addr, err := net.ResolveUDPAddr("udp", *destination)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}

	// create a new gstreamer pipeline.
	var pipelineStr string
	if *test {
		pipelineStr = `
			videotestsrc ! video/x-raw,width=1920,height=1080,framerate=30/1 !
				videoscale ! videorate ! videconvert ! timeoverlay !
				vp8enc error-resilient=1 ! rtpvp8pay pt=96 ! appsink name=videosink
			audiotestsrc ! audioresample ! audio/x-raw,channels=1,rate=48000 !
				opusenc ! rtpopuspay pt=111 ! appsink name=audiosink`
	} else if rtmp != nil {
		// TODO: pipeline string for rtmp.
	}

	pipeline := C.gstreamer_create_pipeline(pipelineStr)

	audioSSRC := uint32(0)
	videoSSRC := uint32(0)

	pipeline.OnAudioPacket(func(buf []byte) {
		p := &rtp.Packet{}
		if err := p.Unmarshal(buf); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal rtp packet")
			return
		}
		audioSSRC = p.SSRC
		if _, err := conn.Write(buf); err != nil {
			panic(err)
		}
	})

	pipeline.OnVideoPacket(func(buf []byte) {
		p := &rtp.Packet{}
		if err := p.Unmarshal(buf); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal rtp packet")
			return
		}
		videoSSRC = p.SSRC
		if _, err := conn.Write(buf); err != nil {
			panic(err)
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