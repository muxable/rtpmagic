package packets

import (
	"strings"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
)

type Codec struct {
	webrtc.RTPCodecCapability
	rtp.Depacketizer

	PayloadType       byte
	GStreamerPipeline string
}

// Type gets the type of codec (video or audio) based on the mime type.
func (c Codec) Type() (webrtc.RTPCodecType, error) {
	if strings.HasPrefix(c.RTPCodecCapability.MimeType, "video") {
		return webrtc.RTPCodecTypeVideo, nil
	} else if strings.HasPrefix(c.RTPCodecCapability.MimeType, "audio") {
		return webrtc.RTPCodecTypeAudio, nil
	}
	return webrtc.RTPCodecType(0), webrtc.ErrUnsupportedCodec
}

// CodecSet is a set of codecs for easy access.
type CodecSet struct {
	byPayloadType map[byte]Codec
}

var defaultCodecSet = NewCodecSet([]Codec{
	// audio codecs
	{
		PayloadType:        111,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		Depacketizer:       &codecs.OpusPacket{},
		GStreamerPipeline:  "opusenc name=audioenc inband-fec=true packet-loss-percentage=10 ! rtpopuspay pt=111",
	},
	// video codecs
	{
		PayloadType:        96,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
		Depacketizer:       &codecs.VP8Packet{},
		GStreamerPipeline:  "vp8enc error-resilient=1 deadline=1 cpu-used=5 keyframe-max-dist=10 auto-alt-ref=1 ! rtpvp8pay pt=96",
	},
	{
		PayloadType:        100,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP9, ClockRate: 90000},
		Depacketizer:       &codecs.VP9Packet{},
		GStreamerPipeline:  "vp9enc error-resilient=1 deadline=1 cpu-used=5 keyframe-max-dist=10 auto-alt-ref=1 ! rtpvp9pay pt=100",
	},
	{
		PayloadType:        102,
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
		Depacketizer:       &codecs.H264Packet{},

	},
})

// NewCodecSet creates a new CodecSet for a given list of codecs.
func NewCodecSet(codecs []Codec) CodecSet {
	set := CodecSet{
		byPayloadType: make(map[byte]Codec),
	}
	for _, codec := range codecs {
		set.byPayloadType[codec.PayloadType] = codec
	}
	return set
}

// FindByPayloadType finds a codec by its payload type.
func (c CodecSet) FindByPayloadType(payloadType byte) (*Codec, bool) {
	codec, ok := c.byPayloadType[payloadType]
	if !ok {
		return nil, false
	}
	return &codec, ok
}

// FindByMimeType finds a codec by its mime type.
func (c CodecSet) FindByMimeType(mimeType string) (*Codec, bool) {
	for _, codec := range c.byPayloadType {
		if codec.RTPCodecCapability.MimeType == mimeType {
			return &codec, true
		}
	}
	return nil, false
}

// DefaultCodecSet gets the default registered codecs.
// These will largely line up with Pion's choices.
func DefaultCodecSet() CodecSet {
	return defaultCodecSet
}
