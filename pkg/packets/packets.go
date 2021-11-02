package packets

import (
	"time"

	"github.com/pion/rtp"
)

type TimestampedPacket struct {
	Timestamp time.Time
	Packet    rtp.Packet
}
