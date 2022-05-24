package ffmpeg

import (
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/muxable/rtpmagic/pkg/muxer/balancer"
	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

type TestSink struct {
	conn *net.UDPConn
}

func NewTestSink(addr string) (*TestSink, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}
	return &TestSink{conn: conn}, nil
}

func (s *TestSink) WriteRTP(p *rtp.Packet) error {
	buf, err := p.Marshal()
	if err != nil {
		return err
	}
	if _, err := s.conn.Write(buf); err != nil {
		return err
	}
	return nil
}

func (s *TestSink) Close() error {
	return s.conn.Close()
}

type BalancerSink struct {
	sources map[uint8]*balancer.ManagedSource
	mpcg   *balancer.ManagedPeerConnectionGroup
}

func NewBalancerSink(params []*webrtc.RTPCodecParameters, sid string, mpcg *balancer.ManagedPeerConnectionGroup) (*BalancerSink, error) {
	// create local tracks.
	sources := make(map[uint8]*balancer.ManagedSource)
	for _, p := range params {
		if p == nil {
			continue
		}
		source, err := mpcg.AddSource(p.RTPCodecCapability, uuid.NewString(), sid)
		if err != nil {
			return nil, err
		}
		sources[uint8(p.PayloadType)] = source
		go rtpio.DiscardRTCP.ReadRTCPFrom(source)
	}

	return &BalancerSink{sources: sources, mpcg: mpcg}, nil
}

func (s *BalancerSink) WriteRTP(p *rtp.Packet) error {
	source, ok := s.sources[p.PayloadType]
	if !ok {
		return fmt.Errorf("no track for payload type %d", p.PayloadType)
	}
	if err := source.WriteRTP(p); err != nil {
		return err
	}
	return nil
}

func (s *BalancerSink) Close() error {
	for _, source := range s.sources {
		if err := s.mpcg.RemoveSource(source); err != nil {
			return err
		}
	}
	return nil
}

var _ rtpio.RTPWriteCloser = (*BalancerSink)(nil)
