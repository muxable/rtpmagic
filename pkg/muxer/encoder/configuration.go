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
					"appsink name=sink async=false sync=false",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1,
			}, nil
		case webrtc.MimeTypeH264:
			return &PipelineConfiguration{
				pipeline: source + " ! nvvidconv interpolation-method=5 ! video/x-raw(memory:NVMM),format=I420 ! " +
					"nvv4l2h264enc bitrate=1000000 preset-level=4 EnableTwopassCBR=true insert-sps-pps=true name=encoder ! " +
					"appsink name=sink async=false sync=false",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1,
			}, nil
		case webrtc.MimeTypeH265:
			return &PipelineConfiguration{
				pipeline: source + " ! nvvidconv interpolation-method=5 ! video/x-raw(memory:NVMM),format=I420 ! " +
					"nvv4l2h265enc bitrate=1000000 preset-level=4 EnableTwopassCBR=true insert-sps-pps=true name=encoder ! " +
					"rtph265pay mtu=1200 pt=106 ! " +
					"appsink name=sink async=false sync=false",
				bitrateProperty: "bitrate",
				bitrateDivisor:  1,
			}, nil
		case webrtc.MimeTypeOpus:
			return &PipelineConfiguration{
				pipeline: source + " ! audioconvert ! " +
					"opusenc inband-fec=true name=encoder ! " +
					"rtpopuspay mtu=1200 pt=111 ! " +
					"appsink name=sink async=false sync=false",
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
