package main

import (
	"flag"
	"os"

	"github.com/muxable/rtpmagic/pkg/muxer"
	"github.com/muxable/rtpmagic/pkg/muxer/encoder"
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

	audioSource, err := enc.AddSource(*audioSrc, audioCodec)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create audio encoder")
	}

	videoSource, err := enc.AddSource(*videoSrc, videoCodec)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create video encoder")
	}

	if err := conn.WriteRTCP([]rtcp.Packet{SourceDescription(*cname, audioSource.SSRC()), SourceDescription(*cname, videoSource.SSRC())}); err != nil {
		log.Fatal().Err(err).Msg("failed to write rtcp packet")
	}

	go rtpio.CopyRTP(conn, audioSource)
	go rtpio.CopyRTP(conn, videoSource)

	// go func() {
	// 	ticker := time.NewTicker(100 * time.Millisecond)
	// 	defer ticker.Stop()
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			if videoSource.ssrc != 0 {
	// 				if err := p.conn.WriteRTCP([]rtcp.Packet{SourceDescription(p.cname, p.videoHandler.ssrc)}); err != nil {
	// 					log.Error().Err(err).Msg("failed to write rtcp")
	// 				}
	// 			}
	// 			if p.audioHandler.ssrc != 0 {
	// 				if err := p.conn.WriteRTCP([]rtcp.Packet{SourceDescription(p.cname, p.audioHandler.ssrc)}); err != nil {
	// 					log.Error().Err(err).Msg("failed to write rtcp")
	// 				}
	// 			}

	// 			// also update the bitrate in this loop because this is a convenient place to do it.
	// 			bitrate, loss := p.conn.GetEstimatedBitrate()
	// 			if bitrate > 64000 {
	// 				bitrate -= 64000 // subtract off audio bitrate
	// 			}
	// 			if bitrate < 100000 {
	// 				bitrate = 100000
	// 			}
	// 			p.video.SetBitrate(p.Pipeline, bitrate)
	// 			C.gstreamer_set_packet_loss_percentage(p.Pipeline, C.guint(loss*100))
	// 		case <-p.ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	select {}
}
