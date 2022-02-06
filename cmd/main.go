package main

import (
	"context"
	"flag"
	"os"

	"github.com/muxable/rtpmagic/pkg/muxer"
	"github.com/muxable/rtpmagic/pkg/muxer/transcoder"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/pion/webrtc/v3"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cname := flag.String("cname", "mugit", "join session name")
	audioSrc := flag.String("audio-src", "alsasrc device=hw:2", "GStreamer audio src")
	videoSrc := flag.String("video-src", "v4l2src", "GStreamer video src")
	netsim := flag.Bool("netsim", false, "enable network simulation")
	dest := flag.String("dest", "34.85.161.200:5000", "rtp sink destination")
	flag.Parse()

	video, err := transcoder.NewPipelineConfiguration(*videoSrc, webrtc.MimeTypeH265)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create video pipeline")
	}
	audio, err := transcoder.NewPipelineConfiguration(*audioSrc, webrtc.MimeTypeOpus)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create audio pipeline")
	}
	codecs := packets.DefaultCodecSet()
	videoCodec, ok := codecs.FindByMimeType(webrtc.MimeTypeH265)
	if !ok {
		log.Fatal().Err(err).Msg("failed to find video codec")
	}
	audioCodec, ok := codecs.FindByMimeType(webrtc.MimeTypeOpus)
	if !ok {
		log.Fatal().Err(err).Msg("failed to find audio codec")
	}

	conn, err := muxer.Dial(*dest, *netsim)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial rtp sink")
	}
	transcoder.CreatePipeline(context.Background(), video, videoCodec, audio, audioCodec, conn, *cname).Start()
}
