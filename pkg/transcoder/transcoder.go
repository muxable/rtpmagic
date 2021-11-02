package transcoder

import (
	"github.com/pion/webrtc/v2/pkg/media"
)

type Transcoder struct {
	in  chan media.Sample
	out chan media.Sample
}

func NewTranscoder(in chan media.Sample, to string) chan media.Sample {
	out := make(chan media.Sample, 16)
	t := &Transcoder{
		in:  in,
		out: out,
	}
	go t.start()
	return out
}

// start starts the transcoding loop.
func (t *Transcoder) start() {
	// start a gst pipeline.
	gst := C.gstreamer_start_pipeline("...")
	// defer its closing.
	defer C.gstreamer_stop_pipeline(gst)

	for p := range t.in {

	}
}
