package main

import (
	"fmt"
	"log"
	"net"

	"github.com/muxable/rtpmagic/pkg/rtmp"
	"github.com/notedit/rtmp-lib/av"
	"github.com/pion/webrtc/v3"
)

func toMimeType(avcodec av.CodecData) (string, error) {
	switch avcodec.Type() {
	case av.H264:
		return webrtc.MimeTypeH264, nil
	case av.AAC:
		return "audio/aac", nil
	default:
		return "", fmt.Errorf("unsupported codec: %v", avcodec.Type())
	}
}

func main() {
	sck, err := net.Listen("tcp", ":1935")
	if err != nil {
		panic(err)
	}
	defer sck.Close()
	for {
		conn, err := sck.Accept()
		if err != nil {
			panic(err)
		}
		rtmpConn := rtmp.Wrap(conn)

		log.Printf("rtmp %v", rtmpConn)
	}
}
