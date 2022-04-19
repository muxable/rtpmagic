package encoder

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0

#include <gst/gst.h>
*/
import "C"
import (
	"fmt"
	"runtime"
	"strings"

	"github.com/pion/webrtc/v3"
)

type PipelineConfiguration struct {
	pipeline        string
	bitrateProperty string
	bitrateDivisor  uint32
}

func NewPipelineConfiguration(source, mimeType string) (*PipelineConfiguration, error) {
	if strings.HasPrefix(runtime.GOARCH, "arm") {
		switch mimeType {
		case webrtc.MimeTypeVP8:
			return &PipelineConfiguration{
				pipeline: source + " ! nvvidconv interpolation-method=5 ! " +
					"nvv4l2vp8enc bitrate=1000000 preset-level=1 name=videoencode ! " +
					"appsink name=sink",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1,
			}, nil
		case webrtc.MimeTypeH264:
			return &PipelineConfiguration{
				pipeline: source + " ! nvvidconv interpolation-method=5 ! video/x-raw(memory:NVMM),format=I420 ! " +
					"nvv4l2h264enc bitrate=1000000 preset-level=4 EnableTwopassCBR=true insert-sps-pps=true name=encoder ! " +
					"rtph264pay mtu=1200 pt=102 config-interval=-1 ! " +
					"appsink name=sink",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1,
			}, nil
		case webrtc.MimeTypeH265:
			return &PipelineConfiguration{
				pipeline: source +
					// " ! video/x-raw,format=(string)YUY2 ! videorate ! video/x-raw,framerate=2997/100 ! queue ! " +
					// "wz265enc preset=ultrafast bitrate=1000 wz265-params=bframes=1:lookahead=2:rc=3:reduce-cplx-tool=1:reduce-cplx-qp=40:fpp=0 key-int-max=240 name=encoder ! queue ! " +
					" ! queue ! videorate ! video/x-raw,framerate=2997/100 ! nvvidconv interpolation-method=5 ! video/x-raw(memory:NVMM),format=I420 ! queue ! " +
					"nvv4l2h265enc bitrate=5000000 preset-level=1 EnableTwopassCBR=true control-rate=1 insert-sps-pps=true name=encoder ! queue ! " +
					"rtph265pay mtu=1200 pt=106 config-interval=1 ! queue ! " +
					"appsink name=sink async=false sync=false",
				bitrateProperty: "bitrate",
				bitrateDivisor: 1,
			}, nil
		case webrtc.MimeTypeOpus:
			return &PipelineConfiguration{
				pipeline: source + " ! queue ! audioconvert ! audioresample ! queue ! " +
					"opusenc inband-fec=true name=encoder bitrate=96000 ! " +
					"rtpopuspay mtu=1200 pt=111 ! queue ! " +
					"appsink name=sink",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1,
			}, nil
		default:
			return nil, fmt.Errorf("unknown mime type")
		}
	} else {
		switch mimeType {
		case webrtc.MimeTypeVP8:
			return &PipelineConfiguration{
				pipeline: source + " ! videoconvert ! " +
					"vp8enc error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 name=videoencode target-bitrate=6000000 ! " +
					"appsink name=videoappsink sync=true",
				bitrateProperty: "target-bitrate",
				bitrateDivisor:  1,
			}, nil
		case webrtc.MimeTypeH264:
			return &PipelineConfiguration{
				pipeline: source + " ! videoconvert ! video/x-raw,format=I420 ! " +
					"x264enc tune=zerolatency bitrate=1000000 ! " +
					"appsink name=sink sync=true",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1000,
			}, nil
		case webrtc.MimeTypeH265:
			return &PipelineConfiguration{
				pipeline: source + " ! videoconvert ! " +
					"x265enc speed-preset=ultrafast tune=zerolatency bitrate=3000 name=encoder ! " +
					"rtph265pay mtu=1200 pt=106 ! " +
					"appsink name=sink sync=false async=false",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1000,
			}, nil
		case webrtc.MimeTypeOpus:
			return &PipelineConfiguration{
				pipeline: source + " ! audioconvert ! " +
					"opusenc inband-fec=true name=encoder ! " +
					"rtpopuspay mtu=1200 pt=111 ! " +
					"appsink name=sink sync=false async=false",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1,
			}, nil
		default:
			return nil, fmt.Errorf("unknown mime type")
		}
	}
}
