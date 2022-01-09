package main

import (
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
	dest := flag.String("dest", "35.245.135.246:5000", "rtp sink destination")
	flag.Parse()

	codecs := packets.DefaultCodecSet()
	videoCodec, _ := codecs.FindByMimeType(webrtc.MimeTypeVP9)
	audioCodec, _ := codecs.FindByMimeType(webrtc.MimeTypeOpus)

	conn, err := muxer.Dial(*dest, *netsim)
	if err != nil {
		panic(err)
	}
	transcoder.CreatePipeline(*videoSrc, videoCodec, *audioSrc, audioCodec, conn, *cname).Start()

	select {}
}
