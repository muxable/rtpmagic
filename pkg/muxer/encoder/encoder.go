package encoder

/*
#cgo pkg-config: gstreamer-1.0

#include <glib.h>
#include <gst/gst.h>

#include "encoder.h"
*/
import "C"
import (
	"errors"
	"runtime"
	"time"
	"unsafe"

	"github.com/mattn/go-pointer"
	"go.uber.org/zap"
)

func init() {
	C.gst_init(nil, nil)
	go C.g_main_loop_run(C.g_main_loop_new(nil, C.int(0)))
}

type Encoder struct {
	bin  *C.GstBin
	ctx  *C.GMainContext
	loop *C.GMainLoop

	closed bool
}

func NewEncoder(cname string) (*Encoder, error) {
	pipeline := C.gst_pipeline_new(nil)

	if C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING) == C.GST_STATE_CHANGE_FAILURE {
		return nil, errors.New("failed to set pipeline to playing")
	}
	ctx := C.g_main_context_new()
	loop := C.g_main_loop_new(ctx, C.int(0))
	watch := C.gst_bus_create_watch(C.gst_pipeline_get_bus((*C.GstPipeline)(unsafe.Pointer(pipeline))))

	s := &Encoder{
		bin:  (*C.GstBin)(unsafe.Pointer(pipeline)),
		ctx:  ctx,
		loop: loop,
	}

	C.g_source_set_callback(watch, C.GSourceFunc(C.cgoBusFunc), C.gpointer(pointer.Save(s)), nil)

	if C.g_source_attach(watch, ctx) == 0 {
		return nil, errors.New("failed to add bus watch")
	}
	defer C.g_source_unref(watch)

	go C.g_main_loop_run(loop)

	runtime.SetFinalizer(s, func(Encoder *Encoder) {
		if err := Encoder.Close(); err != nil {
			zap.L().Error("failed to close Encoder", zap.Error(err))
		}
		C.gst_object_unref(C.gpointer(unsafe.Pointer(Encoder.bin)))
		C.g_main_loop_unref(Encoder.loop)
		C.g_main_context_unref(Encoder.ctx)
	})

	return s, nil
}

func (s *Encoder) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if C.gst_element_set_state((*C.GstElement)(unsafe.Pointer(s.bin)), C.GST_STATE_NULL) == C.GST_STATE_CHANGE_FAILURE {
		return errors.New("failed to set pipeline to null")
	}
	C.g_main_loop_quit(s.loop)
	return nil
}

//export goBusFunc
func goBusFunc(bus *C.GstBus, msg *C.GstMessage, ptr C.gpointer) C.gboolean {
	switch msg._type {
	case C.GST_MESSAGE_ERROR:
		var gerr *C.GError
		var debugInfo *C.gchar
		defer func() {
			if gerr != nil {
				C.g_error_free(gerr)
			}
		}()
		defer C.g_free(C.gpointer(unsafe.Pointer(debugInfo)))
		C.gst_message_parse_error(msg, (**C.GError)(unsafe.Pointer(&gerr)), (**C.gchar)(unsafe.Pointer(&debugInfo)))
		zap.L().Error(C.GoString(gerr.message), zap.String("debug", C.GoString(debugInfo)))
	case C.GST_MESSAGE_WARNING:
		var gerr *C.GError
		var debugInfo *C.gchar
		defer func() {
			if gerr != nil {
				C.g_error_free(gerr)
			}
		}()
		defer C.g_free(C.gpointer(unsafe.Pointer(debugInfo)))
		C.gst_message_parse_warning(msg, (**C.GError)(unsafe.Pointer(&gerr)), (**C.gchar)(unsafe.Pointer(&debugInfo)))
		zap.L().Warn(C.GoString(gerr.message), zap.String("debug", C.GoString(debugInfo)))
	case C.GST_MESSAGE_INFO:
		var gerr *C.GError
		var debugInfo *C.gchar
		defer func() {
			if gerr != nil {
				C.g_error_free(gerr)
			}
		}()
		defer C.g_free(C.gpointer(unsafe.Pointer(debugInfo)))
		C.gst_message_parse_info(msg, (**C.GError)(unsafe.Pointer(&gerr)), (**C.gchar)(unsafe.Pointer(&debugInfo)))
		zap.L().Info(C.GoString(gerr.message), zap.String("debug", C.GoString(debugInfo)))
	case C.GST_MESSAGE_QOS:
		var live C.gboolean
		var runningTime, streamTime, timestamp, duration C.guint64
		C.gst_message_parse_qos(msg, &live, &runningTime, &streamTime, &timestamp, &duration)
		zap.L().Info("QOS",
			zap.Bool("live", live != 0),
			zap.Duration("runningTime", time.Duration(runningTime)),
			zap.Duration("streamTime", time.Duration(streamTime)),
			zap.Duration("timestamp", time.Duration(timestamp)),
			zap.Duration("duration", time.Duration(duration)))
	default:
		zap.L().Debug(C.GoString(C.gst_message_type_get_name(msg._type)), zap.Uint32("seqnum", uint32(msg.seqnum)))
	}
	return C.gboolean(1)
}