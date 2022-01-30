package muxer

import (
	"net"
	"time"

	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/muxer/balancer"
	"github.com/muxable/rtpmagic/test/netsim"
)

type MuxerUDPConn interface {
	rtpio.RTPReadWriteCloser
	rtpio.RTCPReadWriteCloser

	GetEstimatedBitrate() (uint32, float64)
}

func Dial(destination string, useNetsim bool) (MuxerUDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", destination)
	if err != nil {
		return nil, err
	}
	if useNetsim {
		return netsim.NewNetSimUDPConn(addr, []*netsim.ConnectionState{
			{},
		})
	}
	return balancer.NewBalancedUDPConn(addr, 1*time.Second)
}