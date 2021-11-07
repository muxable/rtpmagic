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
	tail                 uint16
	count                uint16
	timestampedPacketIn  chan packets.TimestampedPacket
	timestampedPacketOut chan packets.TimestampedPacket
}

// NewJitterBuffer creates a new singular jitter buffer with the given context and delay.
func NewJitterBuffer(ctx pipeline.Context, delay time.Duration, timestampedPacketIn chan packets.TimestampedPacket) chan packets.TimestampedPacket {
	timestampedPacketOut := make(chan packets.TimestampedPacket)
	buf := &JitterBuffer{
		ctx:                  ctx,
		delay:                delay,
		buffer:               make([]*packets.TimestampedPacket, math.MaxUint16+1),
		timestampedPacketIn:  timestampedPacketIn,
		timestampedPacketOut: timestampedPacketOut,
	}
	go buf.start()
	return timestampedPacketOut
}

func (r *JitterBuffer) start() {
	defer close(r.timestampedPacketOut)
	for {
		if r.count == 0 {
			// the jitterbuffer is empty, so wait for a new packet and insert it directly into the buffer.
			p, ok := <-r.timestampedPacketIn
			if !ok {
				return
			}

			r.buffer[p.Packet.SequenceNumber] = &packets.TimestampedPacket{
				Packet:    p.Packet,
				Timestamp: p.Timestamp.Add(r.delay),
			}

			// set the tail pointer.
			r.tail = p.Packet.SequenceNumber
			r.count++
		} else if t := r.buffer[r.tail]; t != nil {
			select {
			case p, ok := <-r.timestampedPacketIn:
				if !ok {
					return
				}

				// check if the timestamp is to be emitted in the future. if it isn't, then it's too late
				// and emitting it will violate the output invariant.
				emitTimestamp := p.Timestamp.Add(r.delay)
				if emitTimestamp.Before(time.Now()) {
					log.Warn().Msgf("jitterbuffer: packet %d is too late", p.Packet.SequenceNumber)
					break
				}
				// check if this packet is going to be emitted before the tail. if it is, then reset the tail
				// to the incoming packet.
				if r.buffer[p.Packet.SequenceNumber] == nil {
					r.count++
				} else {
					// this is a duplicate packet.
					log.Info().Msgf("duplicate packet %d received", p.Packet.SequenceNumber)
					break
				}
				r.buffer[p.Packet.SequenceNumber] = &packets.TimestampedPacket{
					Packet:    p.Packet,
					Timestamp: emitTimestamp,
				}
				if emitTimestamp.Before(t.Timestamp) || (emitTimestamp == t.Timestamp && p.Packet.SequenceNumber < r.tail) {
					r.tail = p.Packet.SequenceNumber
				}
				break
			case <-r.ctx.Clock.After(t.Timestamp.Sub(r.ctx.Clock.Now())):
				// broadcast the packet at the tail.
				r.timestampedPacketOut <- *t
				r.buffer[r.tail] = nil
				r.tail++
				r.count--
				if r.count > 0 {
					for r.buffer[r.tail] == nil {
						r.tail++
					}
				}
			}
		} else {
			log.Error().Msg("jitter buffer tail is nil but it contains elements! this is an inconsistent state.")
		}
	}
}
