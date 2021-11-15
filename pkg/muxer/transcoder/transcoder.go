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
)

type Transcoder struct {
	mimeType string
	gstElement unsafe.Pointer
	flvIn chan []byte
	rtpOut chan []byte
}

func init() {
	go C.gstreamer_main_loop()
	C.gstreamer_init()
}

func NewTranscoder(mimeType string) (*Transcoder) {
	t := &Transcoder{
		mimeType: mimeType,
	}
	go t.start()
	return t
}

func (t *Transcoder) start() {
	pipelineStr := C.CString(`
		appsrc name=appsrc caps="video/x-raw,format=I420" ! tee name=t
		
		t. decodebin ! vp8enc ! rtpvp8pay pt=96 ! rtpfunnel name=f
		t. decodebin ! opusenc ! rtpopuspay pt=111 ! rtpfunnel name=f

		f. ! appsink name=appsink`)
	defer C.free(unsafe.Pointer(pipelineStr))
	t.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(t))
	for {
		data, ok := <-t.flvIn
		if !ok {
			break
		}
		buf := C.CBytes(data)
		C.gstreamer_write_flv(t.gstElement, buf)
		C.free(buf)
	}
}

//export goHandleRtpAppSinkBuffer
func goHandleRtpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	pipeline := pointer.Restore(data).(*Transcoder)
	pipeline.rtpOut <- C.GoBytes(buffer, C.int(bufferLen))
}


func (t *Transcoder) Read(data []byte) (int, error) {
	buf, ok := <-t.rtpOut
	if !ok {
		return 0, io.EOF
	}
	copy(data, buf)
	return len(buf), nil
}

func (t *Transcoder) Write(data []byte) (int, error) {
	t.flvIn <- data
	return len(data), nil
}

func (t *Transcoder) Close() error {
	close(t.flvIn)
	close(t.rtpOut)
}