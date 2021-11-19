package main

import (
	"fmt"

	"github.com/muxable/rtpmagic/pkg/muxer/transcoder"
	"github.com/notedit/rtmp/av"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

func toMimeType(avcodec int) (string, error) {
	switch avcodec {
	case av.H264:
		return webrtc.MimeTypeH264, nil
	case av.AAC:
		return "audio/aac", nil
	default:
		return "", fmt.Errorf("unsupported codec: %v", avcodec)
	}
}

func main() {
	t := transcoder.NewTranscoder("rtmp://34.72.248.242/live/mugit", "", "")

	for {
		p := &rtp.Packet{}
		_, err := t.ReadRTP(p)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
