package sender

import (
	"github.com/muxable/rtpmagic/pkg/packets"
	sdk "github.com/pion/ion-sdk-go"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
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

func NewRTPSender(addr string, rid string, uid string, codec *packets.Codec, rtpIn chan *rtp.Packet) error {
	c, err := sdk.NewClient(e, addr, uid)
	if err != nil {
		return err
	}

	peerConnection := c.GetPubTransport().GetPeerConnection()

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("Connection state changed: %s", state)
	})

	// Create a video track
	track, err := webrtc.NewTrackLocalStaticRTP(codec.RTPCodecCapability, uid, uid)
	if err != nil {
		return err
	}
	rtpSender, err := peerConnection.AddTrack(track)
	if err != nil {
		return err
	}
	go processRTCP(rtpSender)

	if err := c.Join(rid, sdk.NewJoinConfig().SetNoSubscribe()); err != nil {
		return err
	}

	go func() {
		for p := range rtpIn {
			if err := track.WriteRTP(p); err != nil {
				log.Warn().Err(err).Msg("failed to write sample")
			}
		}
		peerConnection.Close()
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
