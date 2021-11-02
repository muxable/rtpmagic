package jitterbuffer

import (
	"github.com/muxable/rtpmagic/pkg/packets"
)

type NackEmitter struct {
	timestampedPacketIn chan packets.TimestampedPacket
	lastSequenceNumber  uint16
	nackRangeOut        chan []uint16
	initialized         bool // this prevents the first nack emission.
}

// NewNackEmitter creates a new NackEmitter.
func NewNackEmitter(timestampedPacketIn chan packets.TimestampedPacket) chan []uint16 {
	nackRangeOut := make(chan []uint16, 16)
	n := &NackEmitter{
		timestampedPacketIn: timestampedPacketIn,
		nackRangeOut:        nackRangeOut,
	}
	go n.start()
	return nackRangeOut
}

// start starts the reading loop.
func (n *NackEmitter) start() {
	defer close(n.nackRangeOut)
	for tp := range n.timestampedPacketIn {
		if tp.Packet.SequenceNumber == n.lastSequenceNumber+1 {
			n.lastSequenceNumber = tp.Packet.SequenceNumber
		} else {
			if n.initialized {
				n.nackRangeOut <- []uint16{n.lastSequenceNumber + 1, tp.Packet.SequenceNumber - 1}
			}
			n.lastSequenceNumber = tp.Packet.SequenceNumber
		}
		n.initialized = true
	}
}
