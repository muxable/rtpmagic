package normalizer

import (
	"context"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtp"
	log "github.com/rs/zerolog/log"
)

type SSRC uint32

type Session struct {
	initialRTPTimestamp uint32
	initialNTPTimestamp time.Time
	latestNTPTimestamp  time.Time
}

type Normalizer struct {
	ctx               pipeline.Context
	sessions          map[SSRC]*Session
	sessionsLock      sync.Mutex
	sessionExpiration time.Duration
	rtpIn             chan rtp.Packet
	rtpOut            chan packets.TimestampedPacket
}

func NewNormalizer(ctx pipeline.Context, rtpIn chan rtp.Packet) chan packets.TimestampedPacket {
	rtpOut := make(chan packets.TimestampedPacket)
	n := &Normalizer{
		ctx:      ctx,
		sessions: make(map[SSRC]*Session),
		rtpIn:    rtpIn,
		rtpOut:   rtpOut,
	}
	cleanupCtx, cancel := context.WithCancel(context.Background())
	go n.inputLoop(cancel)
	go n.cleanupLoop(cleanupCtx)
	return rtpOut
}

// inputLoop reads values from rtpIn.
func (n *Normalizer) inputLoop(cancel context.CancelFunc) {
	defer cancel()
	for p := range n.rtpIn {
		n.sessionsLock.Lock()
		session := n.sessions[SSRC(p.SSRC)]
		if session == nil {
			// instantiate a new session.
			session = &Session{
				initialRTPTimestamp: p.Timestamp,
				initialNTPTimestamp: n.ctx.Clock.Now(),
			}
			n.sessions[SSRC(p.SSRC)] = session
		}
		n.sessionsLock.Unlock()
		// if this packet is older than the initial timestamp, ignore it.
		//
		// TODO: we should handle this case better because we're assuming that
		// the first packet from a given SSRC is the initialization packet.
		// In reality, this may not be the case if the ordering is incorrect.
		if p.Timestamp < session.initialRTPTimestamp {
			log.Warn().Uint32("Timestamp", p.Timestamp).Uint32("InitialTimestamp", session.initialRTPTimestamp).Msg("received packet with timestamp too early")
			continue
		}
		// get the codec for the given payload type.
		codec, ok := n.ctx.Codecs.FindByPayloadType(p.PayloadType)
		if !ok {
			log.Warn().Uint8("PayloadType", uint8(p.PayloadType)).Uint32("SSRC", p.SSRC).Msg("unknown payload type")
			continue
		}
		// calculate the rtp timestamp delta.
		delta := p.Timestamp - session.initialRTPTimestamp
		// divide by the clock rate to convert to ntp delta.
		ntpDelta := float64(delta) / float64(codec.ClockRate)
		// add the ntpDelta to the session's initial timestamp to compute the effective ntp timestamp.
		ntpTimestamp := session.initialNTPTimestamp.Add(time.Duration(ntpDelta) * time.Second)
		// create a new packet with the new ntp timestamp and write the new packet to the output channel.
		n.rtpOut <- packets.TimestampedPacket{
			Timestamp:    ntpTimestamp,
			Packet:       p,
		}
		// update the latest timestamp for the session for the cleanup loop.
		session.latestNTPTimestamp = ntpTimestamp
	}
}

// cleanupLoop prunes sessions older than sessionExpiration.
func (n *Normalizer) cleanupLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(n.rtpOut)
			return
		case <-n.ctx.Clock.After(1 * time.Second):
			// go through all the sessions and if the latestTimestamp is older than the sessionExpiration,
			// remove the session.
			n.sessionsLock.Lock()
			for ssrc, session := range n.sessions {
				if n.ctx.Clock.Since(session.latestNTPTimestamp) > n.sessionExpiration {
					delete(n.sessions, ssrc)
				}
			}
			n.sessionsLock.Unlock()
		}
	}
}
