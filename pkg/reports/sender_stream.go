package reports

import (
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type SenderStream struct {
	clockRate float64
	m         sync.Mutex

	// data from rtp packets
	lastRTPTimeRTP  uint32
	lastRTPTimeTime time.Time
	packetCount     uint32
	octetCount      uint32
}

func NewSenderStream(clockRate uint32) *SenderStream {
	return &SenderStream{
		clockRate: float64(clockRate),
	}
}

func (stream *SenderStream) ProcessRTP(now time.Time, header *rtp.Header, payload []byte) {
	stream.m.Lock()
	defer stream.m.Unlock()

	// always update time to minimize errors
	stream.lastRTPTimeRTP = header.Timestamp
	stream.lastRTPTimeTime = now

	stream.packetCount++
	stream.octetCount += uint32(len(payload))
}

func (stream *SenderStream) BuildFeedbackPacket(now time.Time, ssrc uint32) *rtcp.SenderReport {
	return &rtcp.SenderReport{
		SSRC:        ssrc,
		NTPTime:     ntpTime(now),
		RTPTime:     stream.lastRTPTimeRTP + uint32(now.Sub(stream.lastRTPTimeTime).Seconds()*stream.clockRate),
		PacketCount: stream.packetCount,
		OctetCount:  stream.octetCount,
	}
}

func ntpTime(t time.Time) uint64 {
	// seconds since 1st January 1900
	s := (float64(t.UnixNano()) / 1000000000) + 2208988800

	// higher 32 bits are the integer part, lower 32 bits are the fractional part
	integerPart := uint32(s)
	fractionalPart := uint32((s - float64(integerPart)) * 0xFFFFFFFF)
	return uint64(integerPart)<<32 | uint64(fractionalPart)
}
