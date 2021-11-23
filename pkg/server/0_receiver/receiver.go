package receiver

import (
	"net"
	"time"

	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/interceptor/v2"
	"github.com/pion/interceptor/v2/pkg/feature"
	"github.com/pion/interceptor/v2/pkg/report"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

// A receiver listens on a UDP port and forwards all incoming RTP packets. RTCP packets are processed by the receiver and are not forwarded.
// Receivers also handle TWCC (Transport Wide Congestion Control) packet generation as well as RTCP packet batching.
func NewReceiver(ctx pipeline.Context, addr string, twccInterval time.Duration) (chan *rtp.Packet, chan *rtcp.Packet, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, nil, err
	}
	log.Debug().Str("addr", udpAddr.String()).Msg("Resolved UDP address")
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, nil, err
	}

	md := feature.NewMediaDescriptionReceiver([]*webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
			PayloadType:        96,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000},
			PayloadType:        111,
		},
	})

	md.WriteMediaDescription(&sdp.MediaDescription{

	})

	rrInterceptor, err := report.NewReceiverInterceptor(md)
	if err != nil {
		return nil, nil, err
	}

	rtpReader, rtcpReader := interceptor.TransformReceiverWithRTCP(NewSSRCManager(ctx, conn), rrInterceptor, 1500)

	rtpOut := make(chan *rtp.Packet)
	rtcpOut := make(chan *rtcp.Packet)

	go func() {
		defer close(rtpOut)
		for  {
			p := &rtp.Packet{}
			if _, err := rtpReader.ReadRTP(p); err != nil {
				return
			}
			rtpOut <- p
		}
	}()

	go func() {
		defer close(rtcpOut)
		for  {
			p := make([]rtcp.Packet, 16)
			if _, err := rtcpReader.ReadRTCP(p); err != nil {
				return
			}
			for _, p2 := range p {
				rtcpOut <- &p2
			}
		}
	}()

	return rtpOut, rtcpOut, nil
}
