package jitterbuffer

import (
	"github.com/muxable/rtpmagic/pkg/packets"
)

type NackEmitter struct {
	timestampedPacketIn chan *packets.TimestampedPacket
	lastSequenceNumber  uint16
	nackOut             chan uint16
	initialized         bool // this prevents the first nack emission.
}

// NewNackEmitter creates a new NackEmitter.
func NewNackEmitter(timestampedPacketIn chan *packets.TimestampedPacket) chan uint16 {
	nackOut := make(chan uint16)
	n := &NackEmitter{
		timestampedPacketIn: timestampedPacketIn,
		nackOut:             nackOut,
	}
	go n.start()
	return nackOut
}

// start starts the reading loop.
func (n *NackEmitter) start() {
	defer close(n.nackOut)
	for tp := range n.timestampedPacketIn {
		if tp.Packet.SequenceNumber == n.lastSequenceNumber+1 {
			n.lastSequenceNumber = tp.Packet.SequenceNumber
		} else {
			if n.initialized {
				for i := n.lastSequenceNumber + 1; i <= tp.Packet.SequenceNumber-1; i++ {
					n.nackOut <- i
				}
			}
			n.lastSequenceNumber = tp.Packet.SequenceNumber
		}
		n.initialized = true
	}
}
