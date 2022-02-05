package transcoder

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0

#include "gst.h"
*/
import "C"
import (
	"context"
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

	video *PipelineConfiguration
	audio *PipelineConfiguration

	ctx context.Context

	videoHandler *SampleHandler
	audioHandler *SampleHandler
}

// CreatePipeline creates a GStreamer Pipeline
func CreatePipeline(
	ctx context.Context,
	video *PipelineConfiguration,
	audio *PipelineConfiguration,
	sink muxer.MuxerUDPConn, cname string) *Pipeline {
	log.Printf("%v", fmt.Sprintf("%s\n%s", video.pipeline, audio.pipeline))

	pipelineStrUnsafe := C.CString(fmt.Sprintf("%s\n%s", video.pipeline, audio.pipeline))
	defer C.free(unsafe.Pointer(pipelineStrUnsafe))

	codecs := packets.DefaultCodecSet()
	videoCodec, ok := codecs.FindByMimeType(webrtc.MimeTypeH264)
	if !ok {
		panic("no codec")
	}
	audioCodec, ok := codecs.FindByMimeType(webrtc.MimeTypeOpus)
	if !ok {
		panic("no codec")
	}

	return &Pipeline{
		Pipeline: C.gstreamer_send_create_pipeline(pipelineStrUnsafe),
		conn:     sink,
		cname:    cname,

		video: video,
		audio: audio,

		ctx: ctx,

		videoHandler: NewSampleHandler(videoCodec),
		audioHandler: NewSampleHandler(audioCodec),
	}
}

func SourceDescription(cname string, ssrc webrtc.SSRC) *rtcp.SourceDescription {
	return &rtcp.SourceDescription{
		Chunks: []rtcp.SourceDescriptionChunk{{
			Source: uint32(ssrc),
			Items:  []rtcp.SourceDescriptionItem{{Type: rtcp.SDESCNAME, Text: cname}},
		}},
	}
}

func (p *Pipeline) writeRTCPLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if p.videoHandler.ssrc != 0 {
				if _, err := p.conn.WriteRTCP([]rtcp.Packet{SourceDescription(p.cname, p.videoHandler.ssrc)}); err != nil {
					log.Error().Err(err).Msg("failed to write rtcp")
				}
			}
			if p.audioHandler.ssrc != 0 {
				if _, err := p.conn.WriteRTCP([]rtcp.Packet{SourceDescription(p.cname, p.audioHandler.ssrc)}); err != nil {
					log.Error().Err(err).Msg("failed to write rtcp")
				}
			}

			// also update the bitrate in this loop because this is a convenient place to do it.
			bitrate, loss := p.conn.GetEstimatedBitrate()
			if bitrate > 64000 {
				bitrate -= 64000 // subtract off audio bitrate
			}
			if bitrate < 100000 {
				bitrate = 100000
			}
			p.video.SetBitrate(p.Pipeline, bitrate)
			C.gstreamer_set_packet_loss_percentage(p.Pipeline, C.guint(loss*100))
		case <-p.ctx.Done():
			return
		}
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
					log.Warn().Uint16("Seq", id).Msg("responding to video nack")
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
			default:
				log.Error().Uint32("SSRC", pkt.MediaSSRC).Msg("nack referring to unknown ssrc")
			}
		}
	}
}

// Start starts the GStreamer Pipeline
func (p *Pipeline) Start() {
	go p.readRTCPLoop()
	go p.writeRTCPLoop()
	C.gstreamer_send_start_pipeline(p.Pipeline, pointer.Save(p))
	<-p.ctx.Done()
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

	p.videoHandler.ssrc = webrtc.SSRC(pkt.SSRC)
	p.videoHandler.sendBuffer.Add(pkt.SequenceNumber, time.Now(), pkt)
	if _, err := p.conn.WriteRTP(pkt); err != nil {
		log.Error().Err(err).Msg("failed to write rtp")
	}
}