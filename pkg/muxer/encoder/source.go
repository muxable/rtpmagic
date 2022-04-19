package encoder

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0 gstreamer-plugins-base-1.0
#cgo LDFLAGS: -lgstsdp-1.0

#include <glib.h>
#include <gst/gst.h>
#include <gst/app/gstappsrc.h>
#include <gst/app/gstappsink.h>
#include <gst/sdp/gstsdpmessage.h>

void X_gst_g_object_set_uint(GstElement *e, const gchar* p_name, guint p_value) {
  g_object_set(G_OBJECT(e), p_name, p_value, NULL);
}
*/
import "C"
import (
	"errors"
	"io"
	"log"
	"time"
	"unsafe"

	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Source struct {
	cfg     *PipelineConfiguration
	element *C.GstElement
	sink    *C.GstAppSink
	encoder *C.GstElement
	sdp     string
	ssrc    webrtc.SSRC
}

var (
	csink = C.CString("sink")
	cclockrate = C.CString("clock-rate")
)

// CreatePipeline creates a GStreamer Pipeline
func (e *Encoder) AddSource(source string, codec *packets.Codec) (*Source, error) {
	p, err := NewPipelineConfiguration(source, codec.MimeType)
	if err != nil {
		return nil, err
	}

	pipelineStr := C.CString(p.pipeline)
	defer C.free(unsafe.Pointer(pipelineStr))

	log.Printf("%v", p.pipeline)

	var gerr *C.GError
	element := C.gst_parse_bin_from_description(pipelineStr, C.int(0), (**C.GError)(&gerr))

	if gerr != nil {
		defer C.g_error_free((*C.GError)(gerr))
		errMsg := C.GoString(gerr.message)
		return nil, errors.New(errMsg)
	}

	if C.gst_bin_add(e.bin, element) == 0 {
		return nil, errors.New("failed to add bin to pipeline")
	}

	if C.gst_element_sync_state_with_parent(element) == 0 {
		return nil, errors.New("failed to sync bin with parent")
	}

	bin := (*C.GstBin)(unsafe.Pointer(element))

	sink := C.gst_bin_get_by_name(bin, csink)

	cencoder := C.CString("encoder")
	defer C.free(unsafe.Pointer(cencoder))

	encoder := C.gst_bin_get_by_name(bin, cencoder)

	pad := C.gst_element_get_static_pad(sink, csink)
	if pad == nil {
		return nil, errors.New("failed to get src pad")
	}
	defer C.gst_object_unref(C.gpointer(pad))

	time.Sleep(1 * time.Second)

	var sdpStr string
	var ssrc webrtc.SSRC
	for {
		caps := C.gst_pad_get_current_caps(pad)
		if caps == nil {
			time.Sleep(1 * time.Millisecond) // it would be nice to not poll for this.
			continue
		}
		defer C.gst_caps_unref(caps)

		ccaps := C.gst_caps_to_string(caps)
		defer C.free(unsafe.Pointer(ccaps))

		log.Printf("caps: %v", C.GoString(ccaps))

		structure := C.gst_caps_get_structure(caps, C.guint(0))

		cssrc := C.CString("ssrc")
		defer C.free(unsafe.Pointer(cssrc))

		var val C.uint

		if C.gst_structure_get_uint(structure, cssrc, &val) == C.gboolean(0) {
			return nil, errors.New("failed to get ssrc")
		}

		ssrc = webrtc.SSRC(val)

		var sdpMedia *C.GstSDPMedia
		if C.gst_sdp_media_new(&sdpMedia) != C.GstSDPResult(0) {
			return nil, errors.New("failed to create sdp media")
		}

		if C.gst_sdp_media_set_media_from_caps(caps, sdpMedia) != C.GstSDPResult(0) {
			return nil, errors.New("failed to set sdp media from caps")
		}

		crtpavp := C.CString("RTP/AVP")
		defer C.free(unsafe.Pointer(crtpavp))

		if C.gst_sdp_media_set_proto(sdpMedia, crtpavp) != C.GstSDPResult(0) {
			return nil, errors.New("failed to set sdp proto")
		}

		csdpStr := C.gst_sdp_media_as_text(sdpMedia)
		defer C.g_free(C.gpointer(csdpStr))

		sdpStr = C.GoString(csdpStr)

		break
	}

	return &Source{
		cfg:     p,
		element: element,
		sink:    (*C.GstAppSink)(unsafe.Pointer(sink)),
		encoder: (*C.GstElement)(unsafe.Pointer(encoder)),
		sdp:     sdpStr,
		ssrc:    ssrc,
	}, nil
}

func (s *Source) SSRC() webrtc.SSRC {
	return s.ssrc
}

func (s *Source) SetBitrate(bitrate uint32) {
	cstr := C.CString(s.cfg.bitrateProperty)
	defer C.free(unsafe.Pointer(cstr))
	C.X_gst_g_object_set_uint(s.encoder, cstr, C.guint(bitrate/s.cfg.bitrateDivisor))
}

func (s *Source) SetPacketLossPercentage(loss uint32) {
	cstr := C.CString("packet-loss-percentage")
	defer C.free(unsafe.Pointer(cstr))
	C.X_gst_g_object_set_uint(s.encoder, cstr, C.guint(loss))
}

func (s *Source) ReadRTP() (*rtp.Packet, error) {
	sample := C.gst_app_sink_pull_sample(s.sink)
	if sample == nil {
		return nil, io.EOF
	}
	defer C.gst_sample_unref(sample)

	cbuf := C.gst_sample_get_buffer(sample)
	if cbuf == nil {
		return nil, io.ErrUnexpectedEOF
	}

	var copy C.gpointer
	var size C.ulong
	C.gst_buffer_extract_dup(cbuf, C.ulong(0), C.gst_buffer_get_size(cbuf), &copy, &size)
	defer C.free(unsafe.Pointer(copy))

	pkt := &rtp.Packet{}
	if err := pkt.Unmarshal(C.GoBytes(unsafe.Pointer(copy), C.int(size))); err != nil {
		return nil, err
	}
	// gstreamer by default uses the pts for timestamp. this is kind of terrible, we use dts.
	pad := C.gst_element_get_static_pad((*C.GstElement)(unsafe.Pointer(s.sink)), csink)
	if pad == nil {
		return nil, errors.New("failed to get src pad")
	}
	caps := C.gst_pad_get_current_caps(pad)
	if caps == nil {
		return nil, errors.New("failed to get caps")
	}
	defer C.gst_caps_unref(caps)

	structure := C.gst_caps_get_structure(caps, C.guint(0))

	var clockrate C.int
	if C.gst_structure_get_int(structure, cclockrate, &clockrate) == C.gboolean(0) {
		return nil, errors.New("failed to get clock rate")
	}
	dts := cbuf.dts
	ts := uint32(float64(dts) * float64(clockrate) / float64(time.Second))
	pkt.Timestamp = ts
	return pkt, nil
}
