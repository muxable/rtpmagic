package jitterbuffer

import (
	"strconv"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/muxable/rtpmagic/pkg/server/2_jitterbuffer/normalizer"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

// NewCompositeJitterBuffer creates a new sequence of jitter buffers with nack emitters in between them.
//
//                  -evict-  -evict-
//                 |       ||       |
// pipes: rtpIn - jb - o - jb - o - jb - rtpOut
//                     |        |
// nack:               nack     nack
//                     |		|
//					    ---------------> nackOut
func NewCompositeJitterBuffer(ctx pipeline.Context, rtpIn chan *rtp.Packet, delays []time.Duration, nackInterval time.Duration) (chan *rtp.Packet, chan []rtcp.NackPair) {
	pipes := make([]chan *packets.TimestampedPacket, len(delays)+1)
	evicts := make([]chan *packets.TimestampedPacket, len(delays))
	nackFunnel := make(chan uint16)
	nackOut := make(chan []rtcp.NackPair)
	var wg sync.WaitGroup

	pipes[0] = normalizer.NewNormalizer(ctx, rtpIn)
	for i, delay := range delays {
		if i == 0 {
			pipes[i+1], evicts[i] = NewJitterBuffer(ctx, strconv.FormatInt(int64(i), 10), delay, pipes[i])
		} else {
			// the input has to be multicasted to a nack emitter too.
			jb := make(chan *packets.TimestampedPacket)
			nack := make(chan *packets.TimestampedPacket)
			go func(i int) {
				for p := range pipes[i] {
					jb <- p
					nack <- p
				}
			}(i)
			// also pipe the evicts from the last jitter buffer, which skip the nack emitter.
			go func(i int) {
				for p := range evicts[i-1] {
					jb <- p
				}
			}(i)
			pipes[i+1], evicts[i] = NewJitterBuffer(ctx, strconv.FormatInt(int64(i), 10), delay, jb)
			nackCh := NewNackEmitter(nack)

			wg.Add(1)
			go func() {
				defer wg.Done()
				for nack := range nackCh {
					nackFunnel <- nack
				}
			}()
		}
	}

	// sink the last evict, these are packets that are too late.
	go func() {
		for range evicts[len(delays)-1] {
		}
	}()

	go func() {
		wg.Wait()
		close(nackFunnel)
	}()

	go func() {
		ticker := time.NewTicker(nackInterval)
		var missing []uint16
		defer ticker.Stop()
		for {
			select {
			case nack, ok := <-nackFunnel:
				if !ok {
					return
				}
				missing = append(missing, nack)
			case <-ticker.C:
				// broadcast the nack packets.
				if len(missing) > 0 {
					nackOut <- rtcp.NackPairsFromSequenceNumbers(missing)
					missing = nil
				}
			}
		}
	}()

	rtpOut := make(chan *rtp.Packet)

	go func() {
		for p := range pipes[len(delays)] {
			rtpOut <- p.Packet
		}
		close(rtpOut)
	}()

	return rtpOut, nackOut
}
