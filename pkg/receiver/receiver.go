package receiver

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/interceptor/pkg/twcc"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

// A receiver listens on a UDP port and forwards all incoming RTP packets. RTCP packets are processed by the receiver and are not forwarded.
// Receivers also handle TWCC (Transport Wide Congestion Control) packet generation as well as RTCP packet batching.
type Receiver struct {
	ctx pipeline.Context

	rtpOut     chan *rtp.Packet
	rtcpOut    chan *rtcp.Packet
	rtcpReturn chan rtcp.CompoundPacket
	conn       *net.UDPConn

	sourcesLock sync.RWMutex
	sources     map[uint32]*net.UDPAddr // maps ssrc to the most recent sender.

	twccRecorders map[int]*twcc.Recorder
	twccInterval  time.Duration
	twccContext   context.Context
	twccCancel    context.CancelFunc
}

func NewReceiver(ctx pipeline.Context, addr string, twccInterval time.Duration, rtcpReturn chan rtcp.CompoundPacket) (chan *rtp.Packet, chan *rtcp.Packet, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, nil, err
	}
	rtpOut := make(chan *rtp.Packet)
	rtcpOut := make(chan *rtcp.Packet)
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, nil, err
	}
	twccContext, twccCancel := context.WithCancel(context.Background())
	r := &Receiver{
		ctx:           ctx,
		rtpOut:        rtpOut,
		rtcpOut:       rtcpOut,
		conn:          conn,
		rtcpReturn:    rtcpReturn,
		sources:       make(map[uint32]*net.UDPAddr),
		twccRecorders: make(map[int]*twcc.Recorder),
		twccInterval:  twccInterval,
		twccContext:   twccContext,
		twccCancel:    twccCancel,
	}
	go r.forward()
	go r.reverse()
	return rtpOut, rtcpOut, nil
}

// forward starts processing incoming packets.
func (r *Receiver) forward() {
	defer close(r.rtpOut)
	defer close(r.rtcpOut)
	for {
		buf := make([]byte, 1500)
		n, sender, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		p := &rtp.Packet{}
		if err := p.Unmarshal(buf[:n]); err != nil {
			log.Warn().Msgf("failed to parse rtp packet: %v", err)
			continue
		}
		if _, ok := r.ctx.Codecs.FindByPayloadType(p.PayloadType); !ok {
			// assume this is an RTCP packet.
			p2, err := rtcp.Unmarshal(buf[:n])
			if err != nil {
				log.Warn().Msgf("failed to parse rtcp packet: %v", err)
				continue
			}
			for _, p3 := range p2 {
				r.rtcpOut <- &p3
			}
		} else {
			r.sourcesLock.Lock()
			r.sources[p.SSRC] = sender
			r.sourcesLock.Unlock()
			t, ok := r.twccRecorders[sender.Port]
			if !ok {
				t = twcc.NewRecorder(r.ctx.SenderSSRC)
				r.twccRecorders[sender.Port] = t
				go func() {
					ticker := time.NewTicker(r.twccInterval)
					for {
						select {
						case <-r.twccContext.Done():
							ticker.Stop()
							return
						case <-ticker.C:
							r.rtcpReturn <- t.BuildFeedbackPacket()
						}
					}
				}()
			}
			t.Record(p.SSRC, p.SequenceNumber, r.ctx.Clock.Now().Unix())
			r.rtpOut <- p
		}
	}
}

// reverse sends RTCP packets to the appropriate destination.
func (r *Receiver) reverse() {
	defer r.conn.Close()
	defer r.twccCancel()
	for p := range r.rtcpReturn {
		// Marshalling this as a CompoundPacket requires receiver reports.
		b, err := rtcp.Marshal([]rtcp.Packet(p))
		if err != nil {
			log.Warn().Msgf("failed to marshal rtcp packet: %v", err)
			continue
		}

		r.sourcesLock.RLock()
		for _, ssrcs := range p.DestinationSSRC() {
			// forward this packet to that ssrc's source.
			if addr, ok := r.sources[ssrcs]; ok {
				if _, err := r.conn.WriteToUDP(b, addr); err != nil {
					log.Error().Err(err).Msg("failed to send rtcp packet")
				}
			}
		}
		r.sourcesLock.RUnlock()
	}
}
