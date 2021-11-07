package receiver

import (
	"net"
	"sync"

	"github.com/muxable/rtpmagic/pkg/pipeline"
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
	rtcpReturn chan *rtcp.Packet
	conn       *net.UDPConn

	sourcesLock sync.RWMutex
	sources     map[uint32]*net.UDPAddr // maps ssrc to the most recent sender.
}

func NewReceiver(ctx pipeline.Context, addr string, rtcpReturn chan *rtcp.Packet) (chan *rtp.Packet, chan *rtcp.Packet, error) {
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
	r := &Receiver{
		ctx:        ctx,
		rtpOut:     rtpOut,
		rtcpOut:    rtcpOut,
		conn:       conn,
		rtcpReturn: rtcpReturn,
		sources:    map[uint32]*net.UDPAddr{},
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
			r.rtpOut <- p
		}
	}
}

// reverse sends RTCP packets to the appropriate destination.
func (r *Receiver) reverse() {
	defer r.conn.Close()
	for p := range r.rtcpReturn {
		b, err := (*p).Marshal()
		if err != nil {
			log.Warn().Msgf("failed to marshal rtcp packet: %v", err)
			continue
		}

		// TODO: batch packets.

		r.sourcesLock.RLock()
		for _, ssrcs := range (*p).DestinationSSRC() {
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
