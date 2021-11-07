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

func (jb *JitterBuffer) start() {
	defer close(jb.timestampedPacketOut)
	for {
		if jb.count == 0 {
			// the jitterbuffer is empty, so wait for a new packet and insert it directly into the buffer.
			p, ok := <-jb.timestampedPacketIn
			if !ok {
				return
			}
			emitTimestamp := p.Timestamp.Add(jb.delay)
			if emitTimestamp.Before(time.Now()) {
				log.Warn().Msgf("jitterbuffer: packet %d is too late", p.Packet.SequenceNumber)
				jb.timestampedPacketOut <- p
				break
			}

			jb.buffer[p.Packet.SequenceNumber] = &packets.TimestampedPacket{
				Packet:    p.Packet,
				Timestamp: emitTimestamp,
			}

			// set the tail pointer.
			jb.tail = p.Packet.SequenceNumber
			jb.count++
		} else if t := jb.buffer[jb.tail]; t != nil {
			select {
			case p, ok := <-jb.timestampedPacketIn:
				if !ok {
					return
				}

				// check if the timestamp is to be emitted in the future. if it isn't, then it's too late
				// and emitting it will violate the output invariant.
				emitTimestamp := p.Timestamp.Add(jb.delay)
				if emitTimestamp.Before(time.Now()) {
					log.Warn().Msgf("jitterbuffer: packet %d is too late", p.Packet.SequenceNumber)
					jb.timestampedPacketOut <- p
					break
				}
				// check if this packet is going to be emitted before the tail. if it is, then reset the tail
				// to the incoming packet.
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
			case <-jb.ctx.Clock.After(t.Timestamp.Sub(jb.ctx.Clock.Now())):
				// broadcast the packet at the tail.
				jb.timestampedPacketOut <- t
				jb.latestTimestamp = t.Timestamp
				jb.buffer[jb.tail] = nil
				jb.tail++
				jb.count--
				if jb.count > 0 {
					for jb.buffer[jb.tail] == nil {
						jb.tail++
					}
				}
			}
		} else {
			log.Error().Msg("jitter buffer tail is nil but it contains elements! this is an inconsistent state.")
		}
	}
}
