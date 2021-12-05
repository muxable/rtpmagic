package packets

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/muxable/rtpio"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Codec struct {
	webrtc.RTPCodecCapability
	webrtc.PayloadType
	Payloader func() rtp.Payloader

	Transcode   func(rtpio.RTPReader) rtpio.RTPReader
	OutputCodec *Codec
}

// Type gets the type of codec (video or audio) based on the mime type.
func (c *Codec) Type() (webrtc.RTPCodecType, error) {
	if strings.HasPrefix(c.RTPCodecCapability.MimeType, "video") {
		return webrtc.RTPCodecTypeVideo, nil
	} else if strings.HasPrefix(c.RTPCodecCapability.MimeType, "audio") {
		return webrtc.RTPCodecTypeAudio, nil
	}
	return webrtc.RTPCodecType(0), webrtc.ErrUnsupportedCodec
}

// Ticker gets a time.Ticker that emits at the frequency of the clock rate.
func (c *Codec) Ticker() *time.Ticker {
	return time.NewTicker(time.Second / time.Duration(c.ClockRate))
}

// CodecSet is a set of codecs for easy access.
type CodecSet struct {
	byPayloadType map[webrtc.PayloadType]Codec
}

var defaultCodecSet = NewCodecSet([]Codec{
	// audio codecs
	{
		PayloadType:        111,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		Payloader: func() rtp.Payloader {
			return &codecs.OpusPayloader{}
		},
	},
	// video codecs
	{
		PayloadType:        96,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
		Payloader: func() rtp.Payloader {
			return &codecs.VP8Payloader{EnablePictureID: true}
		},
	},
	{
		PayloadType:        98,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP9, ClockRate: 90000},
		Payloader: func() rtp.Payloader {
			return &codecs.VP9Payloader{}
		},
	},
	{
		PayloadType:        102,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
		Payloader: func() rtp.Payloader {
			return &codecs.H264Payloader{}
		},
	},
	{
		PayloadType:        106,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/h265", ClockRate: 90000},
		Transcode: func(rtpIn rtpio.RTPReader) rtpio.RTPReader {
			outReader, outWriter := io.Pipe()
			go func() {
				conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 5006})
				if err != nil {
					log.Println(err)
					return
				}
				defer conn.Close()
				for {
					buf := make([]byte, 1024)
					n, _, err := conn.ReadFromUDP(buf)
					if err != nil {
						log.Println(err)
						return
					}
					outWriter.Write(buf[:n])
				}
			}()
			go func() {
				pipeline := ffmpeg.Input("h265transcode.sdp", ffmpeg.KwArgs{
					"protocol_whitelist": "file,crypto,udp,rtp",
				}).Output("rtp://127.0.0.1:5006?pkt_size=1200", ffmpeg.KwArgs{
					"c:v": "libx264",
					"tune": "zerolatency",
					"preset": "ultrafast",
					"pix_fmt": "yuv420p",
					"format": "rtp",
				})
				pipeline.Context = context.WithValue(pipeline.Context, "Stderr", os.Stderr)
				log.Printf("%v", pipeline.GetArgs())
				if err := pipeline.Run(); err != nil {
					log.Printf("%v", err)
				}
			}()
			go func() {
				conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5004})
				if err != nil {
					log.Printf("%v", err)
					return
				}
				for {
					p := &rtp.Packet{}
					if _, err := rtpIn.ReadRTP(p); err != nil {
						return
					}
					if p.Header.PayloadType != 106 {
						continue
					}
					buf, err := p.Marshal()
					if err != nil {
						continue
					}
					if _, err := conn.Write(buf); err != nil {
						return
					}
				}
			}()
			return rtpio.NewRTPReader(outReader, 1500)
		},
		OutputCodec: &Codec{
			PayloadType:        102,
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
			Payloader:          func() rtp.Payloader {
				return &codecs.H264Payloader{}
			},
		},
	},
})

// NewCodecSet creates a new CodecSet for a given list of codecs.
func NewCodecSet(codecs []Codec) *CodecSet {
	set := &CodecSet{
		byPayloadType: make(map[webrtc.PayloadType]Codec),
	}
	for _, codec := range codecs {
		set.byPayloadType[codec.PayloadType] = codec
	}
	return set
}

// FindByPayloadType finds a codec by its payload type.
func (c *CodecSet) FindByPayloadType(payloadType webrtc.PayloadType) (*Codec, bool) {
	codec, ok := c.byPayloadType[payloadType]
	if !ok {
		return nil, false
	}
	return &codec, ok
}

// FindByMimeType finds a codec by its mime type.
func (c *CodecSet) FindByMimeType(mimeType string) (*Codec, bool) {
	for _, codec := range c.byPayloadType {
		if codec.RTPCodecCapability.MimeType == mimeType {
			return &codec, true
		}
	}
	return nil, false
}

// DefaultCodecSet gets the default registered codecs.
// These will largely line up with Pion's choices.
func DefaultCodecSet() *CodecSet {
	return defaultCodecSet
}

func (c *CodecSet) RTPCodecParameters() []*webrtc.RTPCodecParameters {
	var codecs []*webrtc.RTPCodecParameters
	for _, codec := range defaultCodecSet.byPayloadType {
		codecs = append(codecs, &webrtc.RTPCodecParameters{
			RTPCodecCapability: codec.RTPCodecCapability,
			PayloadType:        codec.PayloadType,
		})
	}
	return codecs
}
