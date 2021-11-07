package sender

import (
	sdk "github.com/pion/ion-sdk-go"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
	"github.com/rs/zerolog/log"
)

// new sdk engine
var e = sdk.NewEngine(sdk.Config{
	WebRTC: sdk.WebRTCTransportConfig{
		Configuration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.stunprotocol.org:3478", "stun:stun.l.google.com:19302"},
				},
			},
		},
	},
})

func NewRTPSender(
	addr string, cname string,
	videoCodec webrtc.RTPCodecCapability, audioCodec webrtc.RTPCodecCapability,
	videoIn chan *rtp.Packet, audioIn chan *rtp.Packet) error {
	c, err := sdk.NewClient(e, addr, cname)
	if err != nil {
		return err
	}

	peerConnection := c.GetPubTransport().GetPeerConnection()

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("Connection state changed: %s", state)
	})

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(videoCodec, "video", "pion")
	if err != nil {
		return err
	}
	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return err
	}
	go processRTCP(rtpSender)

	// Create a video track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(audioCodec, "audio", "pion")
	if err != nil {
		return err
	}
	rtpSender, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		return err
	}
	go processRTCP(rtpSender)

	if err := c.Join(cname, sdk.NewJoinConfig().SetNoSubscribe()); err != nil {
		return err
	}

	// Start sending video
	go func() {
		buf := samplebuilder.New(10, &codecs.VP8Packet{}, videoCodec.ClockRate)
		prevSSRC := uint32(0)
		for p := range videoIn {
			if prevSSRC != 0 && prevSSRC != p.Header.SSRC {
				// reset the buffer.
				log.Warn().Uint32("PrevSSRC", prevSSRC).Uint32("NextSSRC", p.Header.SSRC).Str("CNAME", cname).Msg("video sample buffer reset due to SSRC change")
				buf = samplebuilder.New(10, &codecs.VP8Packet{}, videoCodec.ClockRate)
			}
			prevSSRC = p.Header.SSRC
			buf.Push(p)
			for {
				sample := buf.Pop()
				if sample == nil {
					break
				}

				if err := videoTrack.WriteSample(*sample); err != nil {
					log.Warn().Err(err).Msg("failed to write sample")
				}
			}
		}
	}()

	// Start sending audio
	go func() {
		buf := samplebuilder.New(10, &codecs.OpusPacket{}, audioCodec.ClockRate)
		prevSSRC := uint32(0)
		for p := range audioIn {
			if prevSSRC != 0 && prevSSRC != p.Header.SSRC {
				// reset the buffer.
				log.Warn().Uint32("PrevSSRC", prevSSRC).Uint32("NextSSRC", p.Header.SSRC).Str("CNAME", cname).Msg("audio sample buffer reset due to SSRC change")
				buf = samplebuilder.New(10, &codecs.OpusPacket{}, audioCodec.ClockRate)
			}
			prevSSRC = p.Header.SSRC
			buf.Push(p)
			for {
				sample := buf.Pop()
				if sample == nil {
					break
				}

				if err := audioTrack.WriteSample(*sample); err != nil {
					log.Warn().Err(err).Msg("failed to write sample")
				}
			}
		}
	}()

	return nil
}

func processRTCP(rtpSender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)

	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			return
		}
	}
}
