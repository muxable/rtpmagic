package ffmpeg

import (
	"github.com/muxable/rtpmagic/pkg/muxer/balancer"
	"github.com/muxable/sfu/pkg/av"
)

type Encoder struct {
	encoders []*av.EncodeContext
	device   *av.DemuxContext
}

func NewAudioVideoEncoder(
	device *av.DemuxContext,
	audioConfig, videoConfig *av.EncoderConfiguration,
	mpcg *balancer.ManagedPeerConnectionGroup,
	cname string) (*Encoder, error) {

	decoders, err := device.NewDecoders()
	if err != nil {
		return nil, err
	}
	configs, err := decoders.MapEncoderConfigurations(audioConfig, videoConfig)
	if err != nil {
		return nil, err
	}
	encoders, err := decoders.NewEncoders(configs)
	if err != nil {
		return nil, err
	}
	mux, err := encoders.NewRTPMuxer()
	if err != nil {
		return nil, err
	}
	params, err := mux.RTPCodecParameters()
	if err != nil {
		return nil, err
	}
	balancerSink, err := NewBalancerSink(params, cname, mpcg)
	if err != nil {
		return nil, err
	}
	// testSink, err := NewTestSink("100.105.100.81:5000")
	// if err != nil {
	// 	return nil, err
	// }

	// wire them together
	mux.Sink = balancerSink

	return &Encoder{
		encoders: encoders,
		device:   device,
	}, nil
}

func (e *Encoder) SetBitrate(bitrate int64) error {
	effectiveBitrate := bitrate / int64(len(e.encoders))
	for _, encoder := range e.encoders {
		encoder.SetBitrate(effectiveBitrate)
	}
	return nil
}

func (e *Encoder) Run() error {
	return e.device.Run()
}
