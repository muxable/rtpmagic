package packets

import (
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Codec struct {
	webrtc.RTPCodecCapability
	rtp.Depacketizer

	PayloadType byte
}


// CodecSet is a set of codecs for easy access.
type CodecSet struct {
	byPayloadType map[byte]Codec
}

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
