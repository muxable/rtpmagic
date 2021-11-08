package jitterbuffer

import (
	"math"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/rs/zerolog/log"
)

type JitterBuffer struct {
	sync.RWMutex
	ctx                  pipeline.Context
	delay                time.Duration
	buffer               []*packets.TimestampedPacket
	name                 string
	tail                 uint16
	count                uint16
	timestampedPacketIn  chan *packets.TimestampedPacket
	timestampedPacketOut chan *packets.TimestampedPacket
	latestTimestamp      time.Time
}

// NewJitterBuffer creates a new singular jitter buffer with the given context and delay.
func NewJitterBuffer(ctx pipeline.Context, name string, delay time.Duration, timestampedPacketIn chan *packets.TimestampedPacket) chan *packets.TimestampedPacket {
	timestampedPacketOut := make(chan *packets.TimestampedPacket)
	buf := &JitterBuffer{
		ctx:                  ctx,
		delay:                delay,
		name:                 name,
		buffer:               make([]*packets.TimestampedPacket, math.MaxUint16+1),
		timestampedPacketIn:  timestampedPacketIn,
		timestampedPacketOut: timestampedPacketOut,
		latestTimestamp:      ctx.Clock.Now(),
	}
	go buf.start()
	return timestampedPacketOut
}

// getMid returns the index of the next non-nil packet searching from the tail.
func (jb *JitterBuffer) getMid() (*packets.TimestampedPacket, <-chan time.Time) {
	if jb.count == 0 { // this should never happen.
		return nil, make(chan time.Time)
	}
	for i := jb.tail; ; i++ {
		if p := jb.buffer[i]; p != nil {
			return p, jb.ctx.Clock.After(p.Timestamp.Sub(jb.ctx.Clock.Now()))
		}
	}
}

func (jb *JitterBuffer) start() {
	defer close(jb.timestampedPacketOut)
	for {
		// there's at least one packet in the future, so find it and wait for it.
		// this packet is called the mid.
		mid, after := jb.getMid()
		select {
		case p, ok := <-jb.timestampedPacketIn:
			if !ok {
				return
			}

			// check if the timestamp is to be emitted in the future. if it isn't, then it's too late
			// and emitting it will violate the output invariant.
			emitTimestamp := p.Timestamp.Add(jb.delay)
			if emitTimestamp.Before(jb.latestTimestamp) {
				log.Warn().Msgf("jitterbuffer: packet %d is too late", p.Packet.SequenceNumber)
				jb.timestampedPacketOut <- p
				break
			}
			if jb.buffer[p.Packet.SequenceNumber] == nil {
				jb.count++
			} else {
				// this is a duplicate packet.
				log.Info().Msgf("duplicate packet %d received", p.Packet.SequenceNumber)
				break
			}
			jb.buffer[p.Packet.SequenceNumber] = &packets.TimestampedPacket{
				Packet:    p.Packet,
				Timestamp: emitTimestamp,
			}
		case <-after:
			// broadcast the packet at the tail.
			jb.timestampedPacketOut <- mid
			jb.latestTimestamp = mid.Timestamp
			jb.buffer[jb.tail] = nil
			// start future searches after this packet.
			jb.tail = mid.Packet.SequenceNumber + 1
			jb.count--
		}
	}
}
