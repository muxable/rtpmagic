package sender

import (
	"fmt"
	"sync"

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
	addr string, cname string, ssrc uint32,
	videoCodec webrtc.RTPCodecCapability, audioCodec webrtc.RTPCodecCapability,
	videoIn chan *rtp.Packet, audioIn chan *rtp.Packet) error {
	uid := fmt.Sprintf("%s-%d", cname, ssrc)
	c, err := sdk.NewClient(e, addr, uid)
	if err != nil {
		return err
	}

	peerConnection := c.GetPubTransport().GetPeerConnection()

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("Connection state changed: %s", state)
	})

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(videoCodec, fmt.Sprintf("%s-video", uid), uid)
	if err != nil {
		return err
	}
	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return err
	}
	go processRTCP(rtpSender)

	// Create a video track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(audioCodec, fmt.Sprintf("%s-audio", uid), uid)
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

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		rtpToTrack(videoIn, videoTrack, videoCodec)
		wg.Done()
	}()

	go func() {
		rtpToTrack(audioIn, audioTrack, audioCodec)
		wg.Done()
	}()

	go func() {
		wg.Wait()
		peerConnection.Close()
	}()

	return nil
}

func codecToDepacketizer(codec webrtc.RTPCodecCapability) rtp.Depacketizer {
	switch codec.MimeType {
	case webrtc.MimeTypeVP8:
		return &codecs.VP8Packet{}
	case webrtc.MimeTypeVP9:
		return &codecs.VP9Packet{}
	case webrtc.MimeTypeOpus:
		return &codecs.OpusPacket{}
	default:
		log.Warn().Str("MimeType", codec.MimeType).Msg("unsupported codec")
		return nil
	}
}

func rtpToTrack(rtpIn chan *rtp.Packet, track *webrtc.TrackLocalStaticSample, codec webrtc.RTPCodecCapability) {
	buf := samplebuilder.New(10, codecToDepacketizer(codec), codec.ClockRate)
	for p := range rtpIn {
		buf.Push(p)
		for {
			sample := buf.Pop()
			if sample == nil {
				break
			}

			if err := track.WriteSample(*sample); err != nil {
				log.Warn().Err(err).Msg("failed to write sample")
			}
		}
	}
}

func processRTCP(rtpSender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)

	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			return
		}
	}
}
