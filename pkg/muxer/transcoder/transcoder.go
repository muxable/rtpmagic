package transcoder

/*
#cgo CFLAGS: -I/usr/local/lib/x86_64-linux-gnu
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "gstreamer.h"
*/
import "C"
import (
	"io"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/rtp"
)

type Transcoder struct {
	uri           string
	audioMimeType string
	videoMimeType string
	gstElement    *C.GstElement
	rtpOut        chan []byte
}

func init() {
	go C.g_main_loop_run(C.g_main_loop_new(nil, C.int(0)))
	C.gst_init(nil, nil)
}

func NewTranscoder(uri, audioMimeType, videoMimeType string) *Transcoder {
	t := &Transcoder{
		uri:           uri,
		audioMimeType: audioMimeType,
		videoMimeType: videoMimeType,
		rtpOut:        make(chan []byte, 10),
	}
	go t.start()
	return t
}

func (t *Transcoder) start() {
	pipelineStr := C.CString(t.uri)
	defer C.free(unsafe.Pointer(pipelineStr))
	t.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(t))
}

//export goHandleRtpAppSinkBuffer
func goHandleRtpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	t := pointer.Restore(data).(*Transcoder)
	t.rtpOut <- C.GoBytes(buffer, C.int(bufferLen))
}

func (t *Transcoder) ReadRTP(pkt *rtp.Packet) (int, error) {
	buf, ok := <-t.rtpOut
	if !ok {
		return 0, io.EOF
	}
	if err := pkt.Unmarshal(buf); err != nil {
		return 0, err
	}
	return len(buf), nil
}
