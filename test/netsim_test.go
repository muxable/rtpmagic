package normalizer

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtp"
	"go.uber.org/goleak"
)

func TestNetSim_Simple(t *testing.T) {
	in := make(chan rtp.Packet, 10)

	mockClock := clock.New()

	out := NewNormalizer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{
				PayloadType: 96,
				MimeType:    "video",
				ClockRate:   90000,
			},
		}),
		Clock: mockClock,
	}, in)

	p1 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 1,
			Timestamp:      10000,
			PayloadType:    96,
		},
	}

	in <- p1

	mockClock.Sleep(1 * time.Millisecond)

	if len(out) == 0 {
		t.Errorf("expected non-empty out channel")
		return
	}

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val1.Timestamp) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val1.Timestamp))
		return
	}

	close(in)

	goleak.VerifyNone(t)
}