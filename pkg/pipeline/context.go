package pipeline

import (
	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/packets"
)

// Context is a context for a pipeline.
type Context struct {
	Codecs packets.CodecSet
	Clock  clock.Clock
}
