package transcoder

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/nack"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type SampleHandler struct {
	ssrc       webrtc.SSRC
	sendBuffer *nack.SendBuffer
	payloader  rtp.Payloader
	sequencer  rtp.Sequencer
	packetizer rtp.Packetizer
	ClockRate  uint32
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
	if strings.HasPrefix(codec.MimeType, "video/") {
		packetizer := packets.NewTSPacketizer(1200,
			uint8(codec.PayloadType),
			ssrc,
			payloader,
			sequencer,
		)

		return &SampleHandler{
			ssrc:       webrtc.SSRC(ssrc),
			sendBuffer: nack.NewSendBuffer(14),
			payloader:  payloader,
			sequencer:  sequencer,
			packetizer: packetizer,
			ClockRate:  codec.ClockRate,
		}
	} else {
		packetizer := rtp.NewPacketizer(1200,
			uint8(codec.PayloadType),
			ssrc,
			payloader,
			sequencer,
			0,
		)

		return &SampleHandler{
			ssrc:       webrtc.SSRC(ssrc),
			sendBuffer: nack.NewSendBuffer(14),
			payloader:  payloader,
			sequencer:  sequencer,
			packetizer: packetizer,
			ClockRate:  codec.ClockRate,
		}
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

func (h *SampleHandler) packetize(data []byte, samples uint32) []*rtp.Packet {
	return h.packetizer.Packetize(data, samples)
}
