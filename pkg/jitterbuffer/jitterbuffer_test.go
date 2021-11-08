package jitterbuffer

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtp"
	"go.uber.org/goleak"
)

func newJitterBufferPacket(ts time.Time, seq uint16) packets.TimestampedPacket {
	return packets.TimestampedPacket{
		Packet: &rtp.Packet{
			Header: rtp.Header{
				SequenceNumber: seq,
			},
		},
		Timestamp: ts,
	}
}

func assertNoEmissionsForDuration(t *testing.T, ch chan *packets.TimestampedPacket, d time.Duration) {
	select {
	case <-ch:
		t.Errorf("emission received too early")
	case <-time.After(95 * time.Millisecond):
		break
	}
}

func TestJitterBuffer_Simple(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *packets.TimestampedPacket, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewJitterBuffer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{}),
		Clock:  mockClock,
	}, "debug", 100*time.Millisecond, in)

	p1 := newJitterBufferPacket(mockClock.Now(), 100)

	in <- &p1

	select {
	case <-out:
		t.Errorf("emission received too early")
		return
	case <-time.After(99 * time.Millisecond):
		break
	}

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p1, val1)
	}

	if len(out) > 0 {
		t.Errorf("expected empty emit channel")
	}
}

func TestJitterBuffer_TwoPackets_OrdersBySequence(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *packets.TimestampedPacket, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewJitterBuffer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{}),
		Clock:  mockClock,
	}, "debug", 100*time.Millisecond, in)

	ts := mockClock.Now()

	p1 := newJitterBufferPacket(ts, 100)
	p2 := newJitterBufferPacket(ts, 101)

	in <- &p2
	in <- &p1

	assertNoEmissionsForDuration(t, out, 95*time.Millisecond)

	ts1 := time.Now()

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p1, val1)
	}
	val2 := <-out
	if val2.Packet.SequenceNumber != p2.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p2, val2)
	}

	ts2 := time.Now()

	if len(out) > 0 {
		t.Errorf("expected empty emit channel")
	}

	if ts2.Sub(ts1) > 10*time.Millisecond {
		t.Errorf("took too long to emit")
	}
}

func TestJitterBuffer_Deduplicates(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *packets.TimestampedPacket, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewJitterBuffer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{}),
		Clock:  mockClock,
	}, "debug", 100*time.Millisecond, in)

	ts := mockClock.Now()

	p1 := newJitterBufferPacket(ts, 100)
	p2 := newJitterBufferPacket(ts, 101)
	p3 := newJitterBufferPacket(ts, 100)

	in <- &p1
	in <- &p2
	in <- &p3

	assertNoEmissionsForDuration(t, out, 95*time.Millisecond)

	ts1 := time.Now()

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p1, val1)
	}
	val2 := <-out
	if val2.Packet.SequenceNumber != p2.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p2, val2)
	}

	ts2 := time.Now()

	if len(out) > 0 {
		t.Errorf("expected empty emit channel")
	}

	if ts2.Sub(ts1) > 10*time.Millisecond {
		t.Errorf("took too long to emit")
	}
}

func TestJitterBuffer_LotsOfPackets(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *packets.TimestampedPacket, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewJitterBuffer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{}),
		Clock:  mockClock,
	}, "debug", 100*time.Millisecond, in)

	t0 := mockClock.Now()
	t1 := t0.Add(50 * time.Millisecond)

	p1 := newJitterBufferPacket(t0, 101)
	p2 := newJitterBufferPacket(t0, 100)
	p3 := newJitterBufferPacket(t0, 100)
	p4 := newJitterBufferPacket(t0, 103)
	p5 := newJitterBufferPacket(t1, 106)
	p6 := newJitterBufferPacket(t1, 105)
	p7 := newJitterBufferPacket(t1, 104)
	p8 := newJitterBufferPacket(t1, 107)

	in <- &p1
	in <- &p2
	in <- &p3
	in <- &p4
	in <- &p5
	in <- &p6
	in <- &p7
	in <- &p8

	select {
	case <-out:
		t.Errorf("emission received too early")
		return
	case <-time.After(95 * time.Millisecond):
		break
	}

	ts1 := time.Now()

	val1 := <-out
	if val1.Packet.SequenceNumber != p2.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p2, val1)
	}
	val2 := <-out
	if val2.Packet.SequenceNumber != p1.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p1, val2)
	}
	val3 := <-out
	if val3.Packet.SequenceNumber != p4.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p4, val3)
	}

	ts2 := time.Now()

	if len(out) > 0 {
		t.Errorf("expected empty emit channel")
	}

	if ts2.Sub(ts1) > 10*time.Millisecond {
		t.Errorf("took too long to emit")
	}

	mockClock.Sleep(50 * time.Millisecond)

	ts3 := time.Now()

	val4 := <-out
	if val4.Packet.SequenceNumber != p7.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p7, val4)
	}
	val5 := <-out
	if val5.Packet.SequenceNumber != p6.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p6, val5)
	}
	val6 := <-out
	if val6.Packet.SequenceNumber != p5.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p5, val6)
	}
	val7 := <-out
	if val7.Packet.SequenceNumber != p8.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p8, val7)
	}

	ts4 := time.Now()

	if len(out) > 0 {
		t.Errorf("expected empty emit channel")
	}

	if ts4.Sub(ts3) > 10*time.Millisecond {
		t.Errorf("took too long to emit")
	}
}

func TestJitterBuffer_TwoPackets_PacketTooLate(t *testing.T) {
	defer goleak.VerifyNone(t)

	in := make(chan *packets.TimestampedPacket, 10)

	defer close(in)

	mockClock := clock.New()

	out := NewJitterBuffer(pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{}),
		Clock:  mockClock,
	}, "debug", 100*time.Millisecond, in)

	ts := mockClock.Now()

	p1 := newJitterBufferPacket(ts, 102)
	p2 := newJitterBufferPacket(ts.Add(-101*time.Millisecond), 101)

	in <- &p1
	in <- &p2

	select {
	case val2 := <-out:
		// p2 should be evicted immediately.
		if val2.Packet.SequenceNumber != p2.Packet.SequenceNumber {
			t.Errorf("expected %v, got %v", p2, val2)
		}
	case <-time.After(1 * time.Millisecond):
		t.Errorf("expected packet to be evicted")
		break
	}

	assertNoEmissionsForDuration(t, out, 95*time.Millisecond)

	ts1 := time.Now()

	val1 := <-out
	if val1.Packet.SequenceNumber != p1.Packet.SequenceNumber {
		t.Errorf("expected %v, got %v", p1, val1)
	}

	ts2 := time.Now()

	if len(out) > 0 {
		t.Errorf("expected empty emit channel")
	}

	if ts2.Sub(ts1) > 10*time.Millisecond {
		t.Errorf("took too long to emit")
	}
}
