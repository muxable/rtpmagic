package normalizer

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"go.uber.org/goleak"
)

func TestNormalizer_Simple(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *rtp.Packet, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewNormalizer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{
				PayloadType: 96,
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:  "video",
					ClockRate: 90000,
				},
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

	in <- &p1

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val1.Timestamp) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val1.Timestamp))
		return
	}
}

func TestNormalizer_MatchesClockRate(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *rtp.Packet, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewNormalizer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{
				PayloadType: 96,
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:  "video",
					ClockRate: 90000,
				},
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

	p2 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 2,
			Timestamp:      100000,
			PayloadType:    96,
		},
	}

	in <- &p1
	in <- &p2

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal, got %d want %d", val1.Packet.SequenceNumber, p1.SequenceNumber)
		return
	}
	if time.Since(val1.Timestamp) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val1.Timestamp))
		return
	}

	val2 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val2.Timestamp)-(1*time.Second) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val2.Timestamp))
		return
	}
}

func TestNormalizer_SeparateSSRC(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *rtp.Packet, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewNormalizer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{
				PayloadType: 96,
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:  "video",
					ClockRate: 90000,
				},
			},
		}),
		Clock: mockClock,
	}, in)

	p1 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 1,
			Timestamp:      10000,
			PayloadType:    96,
			SSRC:           1,
		},
	}

	p2 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 2,
			Timestamp:      100000,
			PayloadType:    96,
			SSRC:           2,
		},
	}

	in <- &p1
	in <- &p2

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val1.Timestamp) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val1.Timestamp))
		return
	}

	val2 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val2.Timestamp) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val2.Timestamp))
		return
	}
}

func TestNormalizer_CleanupLoop(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *rtp.Packet, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewNormalizer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{
				PayloadType: 96,
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:  "video",
					ClockRate: 90000,
				},
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

	p2 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 2,
			Timestamp:      100000,
			PayloadType:    96,
		},
	}

	in <- &p1

	mockClock.Sleep(1 * time.Second)

	in <- &p2

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val1.Timestamp) > 2*time.Millisecond+1*time.Second {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val1.Timestamp))
		return
	}

	val2 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val2.Timestamp) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val2.Timestamp))
		return
	}
}

func TestNormalizer_TooEarly(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *rtp.Packet, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewNormalizer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{
				PayloadType: 96,
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:  "video",
					ClockRate: 90000,
				},
			},
		}),
		Clock: mockClock,
	}, in)

	p1 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 2,
			Timestamp:      10000,
			PayloadType:    96,
		},
	}

	p2 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 2,
			Timestamp:      0,
			PayloadType:    96,
		},
	}

	in <- &p1
	in <- &p2

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.SequenceNumber {
		t.Errorf("expected packet to be equal")
		return
	}
	if time.Since(val1.Timestamp) > 2*time.Millisecond {
		t.Errorf("expected timestamp to be equal but was %v", time.Since(val1.Timestamp))
		return
	}

	if len(out) != 0 {
		t.Errorf("expected empty out channel")
		return
	}
}

func TestNormalizer_InvalidPayload(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *rtp.Packet, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewNormalizer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{
				PayloadType: 96,
				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:  "video",
					ClockRate: 90000,
				},
			},
		}),
		Clock: mockClock,
	}, in)

	p1 := rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: 2,
			Timestamp:      10000,
			PayloadType:    1,
		},
	}

	in <- &p1

	mockClock.Sleep(1 * time.Millisecond)

	if len(out) != 0 {
		t.Errorf("expected empty out channel")
		return
	}
}
