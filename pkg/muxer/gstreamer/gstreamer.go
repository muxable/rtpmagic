package gstreamer

/*
#cgo CFLAGS: -I/usr/local/lib/x86_64-linux-gnu
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0 glib-2.0
#include "gstreamer.h"
*/
import "C"
import (
	"log"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/rtp"
)

func init() {
	go C.gstreamer_main_loop()
	C.gstreamer_init()
}

type Pipeline struct {
	gstElement *C.GstElement

	pipeline    string
	onRTPPacket func(*rtp.Packet)
}

func NewPipeline(pipeline string) *Pipeline {
	return &Pipeline{
		pipeline: pipeline,
		onRTPPacket: func(p *rtp.Packet) {
			log.Printf("packet %v", p)
		},
	}
}

// OnRTPPacket sets the onRTPPacket callback.
func (p *Pipeline) OnRTPPacket(f func(*rtp.Packet)) *Pipeline {
	p.onRTPPacket = f
	return p
}

func (p *Pipeline) Start() {
	pipelineStr := C.CString(p.pipeline)
	defer C.free(unsafe.Pointer(pipelineStr))
	p.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(p))
}

// Close stops the pipeline.
func (p *Pipeline) Close() {
	C.gstreamer_stop(p.gstElement)
}

//export goHandleRtpAppSinkBuffer
func goHandleRtpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	pipeline := pointer.Restore(data).(*Pipeline)
	p := &rtp.Packet{}
	if err := p.Unmarshal(C.GoBytes(buffer, C.int(bufferLen))); err != nil {
		log.Printf("error unmarshaling packet: %v", err)
		return
	}
	pipeline.onRTPPacket(p)
}
