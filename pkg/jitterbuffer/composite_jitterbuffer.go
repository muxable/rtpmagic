package jitterbuffer

import (
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
)

// NewCompositeJitterBuffer creates a new sequence of jitter buffers with nack emitters in between them.
func NewCompositeJitterBuffer(ctx pipeline.Context, in chan packets.TimestampedPacket, delays []time.Duration) (chan packets.TimestampedPacket, chan []uint16) {
	var out chan packets.TimestampedPacket
	nacks := make(chan []uint16)
	var wg sync.WaitGroup
	for i, delay := range delays {
		out = NewJitterBuffer(ctx, delay, in)

		if i < len(delays)-1 {
			nackOut := NewNackEmitter(out)

			wg.Add(1)

			go func() {
				defer wg.Done()
				for nack := range nackOut {
					nacks <- nack
				}
			}()
		}

		in = out
	}

	go func() {
		wg.Wait()
		close(nacks)
	}()

	return out, nacks
}
