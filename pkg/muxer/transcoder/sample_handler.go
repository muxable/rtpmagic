package transcoder

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/nack"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type SampleHandler struct {
	ssrc       webrtc.SSRC
	sendBuffer *nack.SendBuffer
	payloader  rtp.Payloader
	sequencer  rtp.Sequencer
	packetizer rtp.Packetizer
	clockRate  uint32
}

var ssrcRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func NewSampleHandler(codec *packets.Codec) *SampleHandler {
	if codec.MimeType == "video/h265" {
		return &SampleHandler{
			sendBuffer: nack.NewSendBuffer(14),
		}
	}
	ssrc := ssrcRand.Uint32()
	log.Printf("[rtp] ssrc: %d", ssrc)
	payloader := codec.Payloader()
	sequencer := rtp.NewRandomSequencer()
	packetizer := rtp.NewPacketizer(1200,
		uint8(codec.PayloadType),
		ssrc,
		payloader,
		sequencer,
		codec.ClockRate,
	)

	return &SampleHandler{
		ssrc:       webrtc.SSRC(ssrc),
		sendBuffer: nack.NewSendBuffer(14),
		payloader:  payloader,
		sequencer:  sequencer,
		packetizer: packetizer,
		clockRate:  codec.ClockRate,
	}
}

func (h *SampleHandler) SourceDescription(cname string) *rtcp.SourceDescription {
	return &rtcp.SourceDescription{
		Chunks: []rtcp.SourceDescriptionChunk{{
			Source: uint32(h.ssrc),
			Items:  []rtcp.SourceDescriptionItem{{Type: rtcp.SDESCNAME, Text: fmt.Sprintf("%s-%d", cname, uint32(h.ssrc))}},
		}},
	}
}

func (h *SampleHandler) packetize(sample media.Sample) []*rtp.Packet {
	samples := uint32(time.Duration(sample.Duration).Seconds() * float64(h.clockRate))
	return h.packetizer.Packetize(sample.Data, samples)
}