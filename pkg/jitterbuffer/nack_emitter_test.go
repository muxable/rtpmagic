package jitterbuffer

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/pion/rtp"
	"go.uber.org/goleak"
)

func newNackEmitterPacket(ts time.Time, seq uint16) packets.TimestampedPacket {
	return packets.TimestampedPacket{
		Packet: rtp.Packet{
			Header: rtp.Header{
				SequenceNumber: seq,
			},
		},
		Timestamp: ts,
	}
}

func TestNackEmitter_Simple(t *testing.T) {
	in := make(chan packets.TimestampedPacket, 10)

	mockClock := clock.New()

	out := NewNackEmitter(in)

	ts := mockClock.Now()

	p1 := newNackEmitterPacket(ts, 100)

	in <- p1

	if len(out) > 0 {
		t.Errorf("expected empty out channel")
		return
	}

	close(in)

	goleak.VerifyNone(t)
}

func TestNackEmitter_MissingSingle(t *testing.T) {
	in := make(chan packets.TimestampedPacket, 10)

	mockClock := clock.New()

	out := NewNackEmitter(in)

	ts := mockClock.Now()

	p1 := newNackEmitterPacket(ts, 100)
	p2 := newNackEmitterPacket(ts, 102)

	in <- p1

	if len(out) > 0 {
		t.Errorf("expected empty out channel")
		return
	}

	in <- p2

	mockClock.Sleep(1 * time.Millisecond)

	if len(out) == 0 {
		t.Errorf("expected non-empty out channel")
		return
	}

	val1 := <-out
	if val1[0] != 101 || val1[1] != 101 {
		t.Errorf("expected [101 101], got %v", val1)
		return
	}

	close(in)

	goleak.VerifyNone(t)
}

func TestNackEmitter_MissingRange(t *testing.T) {
	in := make(chan packets.TimestampedPacket, 10)

	mockClock := clock.New()

	out := NewNackEmitter(in)

	ts := mockClock.Now()

	p1 := newNackEmitterPacket(ts, 100)
	p2 := newNackEmitterPacket(ts, 105)

	in <- p1

	if len(out) > 0 {
		t.Errorf("expected empty out channel")
		return
	}

	in <- p2

	mockClock.Sleep(1 * time.Millisecond)

	if len(out) == 0 {
		t.Errorf("expected non-empty out channel")
		return
	}

	val1 := <-out
	if val1[0] != 101 || val1[1] != 104 {
		t.Errorf("expected [101 104], got %v", val1)
		return
	}

	close(in)

	goleak.VerifyNone(t)
}

func TestNackEmitter_MissingTwoBlocks(t *testing.T) {
	in := make(chan packets.TimestampedPacket, 10)

	mockClock := clock.New()

	out := NewNackEmitter(in)

	ts := mockClock.Now()

	p1 := newNackEmitterPacket(ts, 100)
	p2 := newNackEmitterPacket(ts, 105)
	p3 := newNackEmitterPacket(ts, 110)

	in <- p1
	in <- p2
	in <- p3

	mockClock.Sleep(1 * time.Millisecond)

	if len(out) != 2 {
		t.Errorf("expected non-empty out channel")
		return
	}

	val1 := <-out
	if val1[0] != 101 || val1[1] != 104 {
		t.Errorf("expected [101 104], got %v", val1)
		return
	}

	val2 := <-out
	if val2[0] != 106 || val2[1] != 109 {
		t.Errorf("expected [106 109], got %v", val2)
		return
	}

	close(in)

	goleak.VerifyNone(t)
}
