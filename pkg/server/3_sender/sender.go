package sender

import (
	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/packets"
	sdk "github.com/pion/ion-sdk-go"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

func NewRTPSender(addr string, rid string, uid string, codec *packets.Codec, rtpIn rtpio.RTPReader) error {
	connector := sdk.NewConnector(addr)
	rtc := sdk.NewRTC(connector, sdk.DefaultConfig)

	track, err := webrtc.NewTrackLocalStaticRTP(codec.RTPCodecCapability, uid, uid)
	if err != nil {
		return err
	}

	if err := rtc.Join(rid, uid, sdk.NewJoinConfig().SetNoSubscribe()); err != nil {
		return err
	}

	if _, err := rtc.Publish(track); err != nil {
		return err
	}

	for {
		p := &rtp.Packet{}
		if _, err := rtpIn.ReadRTP(p); err != nil {
			return nil
		}
		if err := track.WriteRTP(p); err != nil {
			log.Warn().Err(err).Msg("failed to write sample")
		}
	}
}
