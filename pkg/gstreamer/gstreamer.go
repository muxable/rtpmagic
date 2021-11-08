package gstreamer

/*
#cgo CFLAGS: -I/usr/local/lib/x86_64-linux-gnu
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0 glib-2.0
#include "gst.h"
*/
import "C"
import (
	"log"
	"math"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

func init() {
	go C.gstreamer_main_loop()
	C.gstreamer_init()
}

type Pipeline struct {
	gstElement *C.GstElement

	onAudioPacket func(*rtp.Packet)
	onVideoPacket func(*rtp.Packet)
}

func NewPipeline(s string) *Pipeline {
	return &Pipeline{
		onAudioPacket: func(p *rtp.Packet) {
			log.Printf("audio packet %v", p)
		},
		onVideoPacket: func(p *rtp.Packet) {
			log.Printf("video packet %v", p)
		},
	}
}

// OnAudioPacket sets the onAudioPacket callback.
func (p *Pipeline) OnAudioPacket(f func(*rtp.Packet)) {
	p.onAudioPacket = f
}

// OnVideoPacket sets the onVideoPacket callback.
func (p *Pipeline) OnVideoPacket(f func(*rtp.Packet)) {
	p.onVideoPacket = f
}

func (p *Pipeline) Start(source string, cname string) {
	pipelineStr := C.CString(`
		rtpbin name=rtpbin rtp-profile=avpf sdes="application/x-rtp-source-sdes,cname=(string)\"` + cname + `\""
		
		rtmpsrc location="` + source + ` live=1" ! flvdemux name=demux

		demux.video ! queue ! h264parse ! decodebin ! videoconvert ! nvh265enc bitrate=5000 preset=4 rc-mode=2 zerolatency=true ! h265parse config-interval=-1 ! mux.
		demux.audio ! queue ! aacparse ! decodebin ! audioconvert ! opusenc ! opusparse ! mux.
		
		mpegtsmux name=mux ! rtpmp2tpay pt=33 ! rtprtxqueue ! rtpbin.send_rtp_sink_0
		rtpbin.send_rtp_src_0 ! appsink name=rtpsink
		rtpbin.send_rtcp_src_0 ! appsink name=rtcpsink sync=false async=false
		appsrc name=rtcpsrc1 ! rtpbin.recv_rtcp_sink_0`)
	defer C.free(unsafe.Pointer(pipelineStr))

	log.Printf("starting pipeline %s -> %s", source, cname)

	p.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(p))
}

// WriteRTCP writes an RTCP reply to the session.
func (p *Pipeline) WriteRTCP(buffer []byte) error {
	b := C.CBytes(buffer)
	defer C.free(b)

	cp, err := rtcp.Unmarshal(buffer)
	if err != nil {
		return err
	}
	for _, p := range cp {
		log.Printf("%v", p)
	}

	C.gstreamer_push_rtcp(p.gstElement, b, C.int(len(buffer)))
	return nil
}

//export goHandleRtpAppSinkBuffer
func goHandleRtpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	pipeline := pointer.Restore(data).(*Pipeline)
	if pipeline.PacketSink != nil {
		log.Printf("writing packet %d", bufferLen)
		if _, err := pipeline.PacketSink(C.GoBytes(buffer, C.int(bufferLen))); err != nil {
			log.Printf("error writing to pipeline sink: %v", err)
		}
	}
}

var bitrate = uint64(6000000)
var state = int64(0)

const maxBitrate = uint64(10000000)
const minBitrate = uint64(100000)

//export goHandleTwccStats
func goHandleTwccStats(bitrateSent C.uint, bitrateRecv C.uint, packetsSent C.uint, packetsRecv C.uint, avgDeltaOfDelta C.long, data unsafe.Pointer) {
	p := pointer.Restore(data).(*Pipeline)
	state = state + (int64(avgDeltaOfDelta)-state)/20
	log.Printf("sent %d recv %d pkt sent %d pkt recv %d delta of delta %d state %d", bitrateSent, bitrateRecv, packetsSent, packetsRecv, avgDeltaOfDelta, state)
	// calculate the magnitude of the kalman-filtered state.
	mag := math.Log(math.Abs(float64(state)))
	// log.Printf("magnitude %d", mag)
	if mag > 40 {
		// overuse
		bitrate = uint64(math.Max(float64(bitrate)*0.7, float64(minBitrate)))
		C.gstreamer_set_bitrate(p.gstElement, C.uint(bitrate))
		log.Printf("bitrate %d", bitrate)
	} else if mag < 20 {
		// underuse
		bitrate = uint64(math.Min(float64(bitrate)*1.08, float64(maxBitrate)))
		C.gstreamer_set_bitrate(p.gstElement, C.uint(bitrate))
		log.Printf("bitrate %d", bitrate)
	}
}
