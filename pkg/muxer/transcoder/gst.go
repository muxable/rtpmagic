package transcoder

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0

#include "gst.h"
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/muxable/rtpmagic/pkg/muxer"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

func init() {
	go C.g_main_loop_run(C.g_main_loop_new(nil, C.int(0)))
}

type Pipeline struct {
	Pipeline *C.GstElement
	conn     muxer.MuxerUDPConn
	cname    string

	videoHandler *SampleHandler
	audioHandler *SampleHandler
}

func videoPipelineStr(videoSrc, mimeType string) string {
	switch mimeType {
	case webrtc.MimeTypeVP8:
		return videoSrc + " ! nvvidconv interpolation-method=5 ! nvv4l2vp8enc bitrate=1000000 maxperf-enable=true preset-level=1 name=videoencode ! appsink name=videoappsink"
	case webrtc.MimeTypeVP9:
		return videoSrc + " ! vp9enc deadline=1 ! appsink name=videoappsink"
	case webrtc.MimeTypeH264:
		return videoSrc + " ! nvvidconv interpolation-method=5 ! video/x-raw(memory:NVMM),format=I420 ! nvv4l2h264enc bitrate=1000000 qp-range=\"28,50:0,38:0,50\" iframeinterval=60 preset-level=1 maxperf-enable=true EnableTwopassCBR=true insert-sps-pps=true name=videoencode ! video/x-h264,stream-format=byte-stream ! appsink name=videoappsink"
	case "video/h265":
		return videoSrc + " ! nvvidconv interpolation-method=5 ! video/x-raw(memory:NVMM),format=I420 ! nvv4l2h265enc bitrate=1000000 qp-range=\"28,50:0,38:0,50\" iframeinterval=60 preset-level=1 maxperf-enable=true EnableTwopassCBR=true insert-sps-pps=true name=videoencode ! video/x-h265,stream-format=byte-stream ! rtph265pay pt=106 mtu=1200 ! appsink name=videortpsink"
	default:
		panic("unknown mime type")
	}
}

func audioPipelineStr(audioSrc, mimeType string) string {
	switch mimeType {
	case webrtc.MimeTypeOpus:
		return audioSrc + " ! audioconvert ! opusenc ! appsink name=audioappsink"
	default:
		panic("unknown mime type")
	}
}

// CreatePipeline creates a GStreamer Pipeline
func CreatePipeline(
	videoSrc string, videoCodec *packets.Codec,
	audioSrc string, audioCodec *packets.Codec,
	sink muxer.MuxerUDPConn, cname string) *Pipeline {
	videoPipelineStr := videoPipelineStr(videoSrc, videoCodec.MimeType)
	audioPipelineStr := audioPipelineStr(audioSrc, audioCodec.MimeType)

	pipelineStrUnsafe := C.CString(fmt.Sprintf("%s\n%s", videoPipelineStr, audioPipelineStr))
	defer C.free(unsafe.Pointer(pipelineStrUnsafe))
	
	return &Pipeline{
		Pipeline: C.gstreamer_send_create_pipeline(pipelineStrUnsafe),
		conn:     sink,
		cname:    cname,

		videoHandler: NewSampleHandler(videoCodec),
		audioHandler: NewSampleHandler(audioCodec),
	}
}

func (p *Pipeline) writeRTCPLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		videoSdes := p.videoHandler.SourceDescription(p.cname)
		audioSdes := p.audioHandler.SourceDescription(p.cname)
		if _, err := p.conn.WriteRTCP([]rtcp.Packet{videoSdes, audioSdes}); err != nil {
			log.Error().Err(err).Msg("failed to write rtcp")
			return
		}

		// also update the bitrate in this loop because this is a convenient place to do it.
		bitrate := p.conn.GetEstimatedBitrate()
		if bitrate > 64000 {
			bitrate -= 64000 // subtract off audio bitrate
		}
		if bitrate < 300000 {
			bitrate = 300000
		}
		C.gstreamer_set_video_bitrate(p.Pipeline, C.guint(bitrate))
	}
}

func (p *Pipeline) readRTCPLoop() {
	for {
		pkts := make([]rtcp.Packet, 16)
		n, err := p.conn.ReadRTCP(pkts)
		if err != nil {
			log.Warn().Err(err).Msg("connection error")
			return
		}
		for _, pkt := range pkts[:n] {
			switch pkt := pkt.(type) {
			case *rtcp.PictureLossIndication:
				log.Info().Msg("PLI")
			case *rtcp.ReceiverReport:
				log.Info().Msg("Receiver Report")
			case *rtcp.Goodbye:
				log.Info().Msg("Goodbye")
			case *rtcp.TransportLayerNack:
				p.handleNack(pkt)
			case *rtcp.TransportLayerCC:
				log.Info().Msg("Transport Layer CC")
			default:
				// log.Info().Msgf("unknown rtcp packet: %v", pkt)
			}
		}
	}
}

// handleNack handles a rtcp.TransportLayerNack from the receiver.
func (p *Pipeline) handleNack(pkt *rtcp.TransportLayerNack) {
	for _, nack := range pkt.Nacks {
		for _, id := range nack.PacketList() {
			switch webrtc.SSRC(pkt.MediaSSRC) {
			case p.videoHandler.ssrc:
				if _, q := p.videoHandler.sendBuffer.Get(id); q != nil {
					if _, err := p.conn.WriteRTP(q); err != nil {
						log.Error().Err(err).Msg("failed to write rtp")
					}
				} else {
					log.Warn().Uint16("Seq", id).Msg("nack referring to missing packet")
				}
			case p.audioHandler.ssrc:
				if _, q := p.audioHandler.sendBuffer.Get(id); q != nil {
					if _, err := p.conn.WriteRTP(q); err != nil {
						log.Error().Err(err).Msg("failed to write rtp")
					}
				} else {
					log.Warn().Uint16("Seq", id).Msg("nack referring to missing packet")
				}
			}
		}
	}
}

// Start starts the GStreamer Pipeline
func (p *Pipeline) Start() {
	go p.readRTCPLoop()
	go p.writeRTCPLoop()
	C.gstreamer_send_start_pipeline(p.Pipeline, pointer.Save(p))
}

// Stop stops the GStreamer Pipeline
func (p *Pipeline) Stop() {
	C.gstreamer_send_stop_pipeline(p.Pipeline)
}

//export goHandleAudioPipelineBuffer
func goHandleAudioPipelineBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.ulong, dts C.ulong, data unsafe.Pointer) {
	p := pointer.Restore(data).(*Pipeline)

	samples := uint32(time.Duration(duration).Seconds() * float64(p.audioHandler.ClockRate))

	for _, pkt := range p.audioHandler.packetize(C.GoBytes(buffer, bufferLen), samples) {
		p.audioHandler.sendBuffer.Add(pkt.SequenceNumber, time.Now(), pkt)
		if _, err := p.conn.WriteRTP(pkt); err != nil {
			log.Error().Err(err).Msg("failed to write rtp")
		}
	}
}

//export goHandleVideoPipelineBuffer
func goHandleVideoPipelineBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.ulong, dts C.ulong, data unsafe.Pointer) {
	p := pointer.Restore(data).(*Pipeline)

	rtpts := uint32(uint64(dts) / 1000 * (uint64(p.videoHandler.ClockRate) / 1000) / 1000)

	for _, pkt := range p.videoHandler.packetize(C.GoBytes(buffer, bufferLen), rtpts) {
		p.videoHandler.sendBuffer.Add(pkt.SequenceNumber, time.Now(), pkt)
		if _, err := p.conn.WriteRTP(pkt); err != nil {
			log.Error().Err(err).Msg("failed to write rtp")
		}
	}
}

//export goHandleVideoPipelineRtp
func goHandleVideoPipelineRtp(buffer unsafe.Pointer, bufferLen C.int, duration C.ulong, data unsafe.Pointer) {
	p := pointer.Restore(data).(*Pipeline)

	pkt := &rtp.Packet{}
	if err := pkt.Unmarshal(C.GoBytes(buffer, bufferLen)); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal rtp")
		return
	}

	p.videoHandler.sendBuffer.Add(pkt.SequenceNumber, time.Now(), pkt)
	if _, err := p.conn.WriteRTP(pkt); err != nil {
		log.Error().Err(err).Msg("failed to write rtp")
	}
}
