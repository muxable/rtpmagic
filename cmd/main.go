package main

import (
	"flag"
	"os"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/balancer"
	"github.com/muxable/rtpmagic/pkg/muxer/ffmpeg"
	"github.com/muxable/sfu/pkg/av"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/pion/webrtc/v3"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cname := flag.String("cname", "mugit", "join session name")
	audioSrc := flag.String("audio-src", "plughw:CARD=RX", "audio src")
	videoSrc := flag.String("video-src", "/dev/video0", "video src")
	dest := flag.String("dest", "100.105.100.81:50051", "rtp sink destination")
	flag.Parse()

	audio, err := av.NewDeviceDemuxer("alsa", *audioSrc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create audio device")
	}

	video, err := av.NewDeviceDemuxer("v4l2", *videoSrc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create video device")
	}

	mpcg, err := balancer.NewManagedPeerConnection(*dest, 1 * time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create managed peer connection")
	}

	audioBitrate := int64(96000)
	minimumBitrate := int64(500000)

	audioCodec := &av.EncoderConfiguration{
		Name: "libopus",
		Codec: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels: 2,
		},
		Bitrate: audioBitrate,
	}

	videoCodec :=  &av.EncoderConfiguration{
		// Name: "libwz265",
		Name: "hevc_nvv4l2",
		Codec: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeH265,
			ClockRate: 90000,
		},
		Bitrate: minimumBitrate,
		FrameRateNumerator: 7013,
		FrameRateDenominator: 234,
		Options: map[string]interface{}{
			// "preset": "ultrafast",
			// "wz265-params": "preset=ultrafast:bframes=15:lookahead=2:rc=3:reduce-cplx-tool=1:reduce-cplx-qp=51:fpp=0",
			"preset": "4",
			"2pass": "1",
			"g": "10",
		},
	}

	audioEncoder, err := ffmpeg.NewAudioVideoEncoder(audio, audioCodec, videoCodec, mpcg, *cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create encoder")
	}

	videoEncoder, err := ffmpeg.NewAudioVideoEncoder(video, audioCodec, videoCodec, mpcg, *cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create encoder")
	}

	go audioEncoder.Run()
	go videoEncoder.Run()

	ticker := time.NewTicker(15 * time.Millisecond) // ~60 fps
	for range ticker.C {
		bitrate := int64(mpcg.GetEstimatedBitrate())
		if bitrate > audioBitrate {
			bitrate -= audioBitrate // subtract off audio bitrate
		}
		if bitrate < minimumBitrate {
			bitrate = minimumBitrate
		}
		videoEncoder.SetBitrate(bitrate * 6 / 10)
		// audioSource.SetPacketLossPercentage(uint32(loss * 100))
	}
}
