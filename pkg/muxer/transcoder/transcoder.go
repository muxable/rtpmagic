package transcoder

/*
#cgo CFLAGS: -I/usr/local/lib/x86_64-linux-gnu
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "gstreamer.h"
*/
import "C"
import (
	"log"
	"strings"
	"time"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/pion/rtp"
)

type Transcoder struct {
	uri        string
	audio      *packets.Codec
	video      *packets.Codec
	gstElement *C.GstElement
	RTPOut     chan *rtp.Packet
	bitrate    uint32
}

func init() {
	go C.g_main_loop_run(C.g_main_loop_new(nil, C.int(0)))
	C.gst_init(nil, nil)
}

func NewTranscoder(uri string, audio, video *packets.Codec) *Transcoder {
	t := &Transcoder{
		uri:    uri,
		audio:  audio,
		video:  video,
		RTPOut: make(chan *rtp.Packet, 10),
	}
	go t.start()
	return t
}

func isJetsonNano() bool {
	nvvidconv := C.CString("nvvidconv")
	defer C.free(unsafe.Pointer(nvvidconv))
	return C.gst_element_factory_find(nvvidconv) != nil
}

func (t *Transcoder) SetVideoBitrate(bitrate uint32) {
	// we gate bitrate increases because changing the bitrate causes visible artifacts on the gst encoder.
	// smoothing these out allows us to produce fewer artifacts on stream.
	delta := int64(bitrate) - int64(t.bitrate)
	if delta < -50_000 {
		// process decreases exceeding 50kbps
		C.gstreamer_set_video_bitrate(t.gstElement, C.guint32(bitrate))
		log.Printf("bitrate %v\t%v -> %v", delta, t.bitrate, bitrate)
		t.bitrate = bitrate
	} else if delta > 150_000 {
		// process increases exceeding 150kbps
		C.gstreamer_set_video_bitrate(t.gstElement, C.guint32(bitrate))
		log.Printf("bitrate +%v\t%v -> %v", delta, t.bitrate, bitrate)
		t.bitrate = bitrate
	}
}

func (t *Transcoder) getPipelineStr() (string, error) {
	if strings.HasPrefix(t.uri, "rtmp://") || strings.HasPrefix(t.uri, "testbin://") {
		if isJetsonNano() {
			return `uridecodebin uri="` + t.uri + `" name=demux
				demux. ! queue ! audioconvert !
					opusenc inband-fec=true packet-loss-percentage=8 !
					rtpopuspay pt=111 ! appsink name=audio_sink
				demux. ! queue ! videoconvert !
					nvv4l2vp8enc maxperf-enable=true preset-level=1 error-resilient=2 keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 name=video_encode !
					rtpvp8pay pt=96 ! appsink name=video_sink`, nil
		} else {
			// this is mostly here for debugging.
			return `uridecodebin uri="` + t.uri + `" name=demux
				demux. ! queue ! audioconvert !
					opusenc inband-fec=true packet-loss-percentage=8 !
					rtpopuspay pt=111 ! appsink name=audio_sink
				demux. ! queue ! videoconvert !
					vp8enc error-resilient=2 keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 name=video_encode !
					rtpvp8pay pt=96 ! appsink name=video_sink`, nil
		}
	} else if strings.HasPrefix(t.uri, "v4l2://") {
		deviceName := strings.TrimPrefix(t.uri, "v4l2://")
		if deviceName == "" {
			deviceName = "/dev/video0"
		}
		return `
			alsasrc device=hw:2 ! audioconvert !
				opusenc inband-fec=true packet-loss-percentage=8 !
				rtpopuspay pt=111 ! appsink name=audio_sink
			v4l2src device="` + deviceName + `" ! videoconvert !
				vp8enc error-resilient=2 keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 name=video_encode !
				rtpvp8pay pt=96 ! appsink name=video_sink`, nil
	}
	return "", nil
}

func (t *Transcoder) start() {
	p, err := t.getPipelineStr()
	if err != nil {
		log.Printf("error creating pipeline: %v", err)
		return
	}

	cp := C.CString(p)
	defer C.free(unsafe.Pointer(cp))
	t.gstElement = C.gstreamer_start(cp, pointer.Save(t))
}

//export goHandleRtpAppSinkBuffer
func goHandleRtpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	t := pointer.Restore(data).(*Transcoder)
	p := &rtp.Packet{}
	if err := p.Unmarshal(C.GoBytes(buffer, C.int(bufferLen))); err != nil {
		log.Printf("unmarshal error: %v", err)
	}
	t.RTPOut <- p
}

func toSamples(duration int, clockRate uint32) uint32 {
	t := time.Duration(duration) * time.Nanosecond
	return uint32(t.Seconds()*float64(clockRate) + 0.5)
}
