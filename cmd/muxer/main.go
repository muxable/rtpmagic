package main

import (
	"flag"
	"net"
	"sync"
	"time"

	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/muxer/balancer"
	"github.com/muxable/rtpmagic/pkg/muxer/nack"
	"github.com/muxable/rtpmagic/pkg/muxer/transcoder"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/reports"
	"github.com/muxable/rtpmagic/test/netsim"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type MuxerUDPConn interface {
	rtpio.RTPReadWriteCloser
	rtpio.RTCPReadWriteCloser

	GetEstimatedBitrate() uint32
}

func dial(destination string, useNetsim bool) (MuxerUDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", destination)
	if err != nil {
		return nil, err
	}
	if useNetsim {
		return netsim.NewNetSimUDPConn(addr, []*netsim.ConnectionState{
			{},
		})
	}
	return balancer.NewBalancedUDPConn(addr, 1*time.Second)
}

// the general pipeline is
// audio/video src as string -> raw
// encode audio and video
// broadcast rtp
func main() {
	uri := flag.String("uri", "testbin://audio+video", "source uri")
	cname := flag.String("cname", "mugit", "cname to send as")
	netsim := flag.Bool("netsim", false, "use netsim connection")
	destination := flag.String("dest", "localhost:5000", "destination")
	audioMimeType := flag.String("audio", webrtc.MimeTypeOpus, "audio mime type")
	videoMimeType := flag.String("video", webrtc.MimeTypeVP8, "video mime type")
	flag.Parse()

	audioCodec, ok := packets.DefaultCodecSet().FindByMimeType(*audioMimeType)
	if !ok {
		log.Fatal().Msg("no audio codec")
	}
	videoCodec, ok := packets.DefaultCodecSet().FindByMimeType(*videoMimeType)
	if !ok {
		log.Fatal().Msg("no video codec")
	}

	conn, err := dial(*destination, *netsim)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial")
	}

	pipeline, err := NewPipeline(conn, *uri, audioCodec, videoCodec, *cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create pipeline")
	}

	done := make(chan bool)
	go func() {
		for {
			time.Sleep(250 * time.Millisecond)
			targetBitrate := conn.GetEstimatedBitrate()
			// subtract off 64 kbps because it's reserved for audio.
			pipeline.SetVideoBitrate(targetBitrate - 64000)
		}
	}()

	pipeline.Start()
	done <- true
}

type Pipeline struct {
	sync.RWMutex
	conn              MuxerUDPConn
	transcoder        *transcoder.Transcoder
	audioSSRC         webrtc.SSRC
	videoSSRC         webrtc.SSRC
	audioCodec        *packets.Codec
	videoCodec        *packets.Codec
	audioSendBuffer   *nack.SendBuffer
	videoSendBuffer   *nack.SendBuffer
	audioSenderStream *reports.SenderStream
	videoSenderStream *reports.SenderStream
	cname             string
}

func NewPipeline(conn MuxerUDPConn, uri string, audioCodec *packets.Codec, videoCodec *packets.Codec, cname string) (*Pipeline, error) {
	p := &Pipeline{
		conn:              conn,
		transcoder:        transcoder.NewTranscoder(uri, audioCodec, videoCodec),
		audioCodec:        audioCodec,
		videoCodec:        videoCodec,
		audioSendBuffer:   nack.NewSendBuffer(12),
		videoSendBuffer:   nack.NewSendBuffer(14),
		audioSenderStream: reports.NewSenderStream(audioCodec.ClockRate),
		videoSenderStream: reports.NewSenderStream(videoCodec.ClockRate),
		cname: cname,
	}

	go p.writeRTCPLoop()
	go p.rtcpLoop()

	return p, nil
}

func (p *Pipeline) SetVideoBitrate(bitrate uint32) {
	p.RLock()
	defer p.RUnlock()

	p.transcoder.SetVideoBitrate(bitrate)
}

func (p *Pipeline) Start() {
	for {
		pkt := <-p.transcoder.RTPOut
		switch webrtc.PayloadType(pkt.PayloadType) {
		case p.videoCodec.PayloadType:
			p.videoSSRC = webrtc.SSRC(pkt.SSRC)
			p.videoSendBuffer.Add(pkt.SequenceNumber, time.Now(), pkt)
		case p.audioCodec.PayloadType:
			p.audioSSRC = webrtc.SSRC(pkt.SSRC)
			p.audioSendBuffer.Add(pkt.SequenceNumber, time.Now(), pkt)
		}
		if _, err := p.conn.WriteRTP(pkt); err != nil {
			log.Error().Err(err).Msg("failed to write rtp")
		}
	}
}

func (p *Pipeline) writeRTCPLoop() {
	// send forward SDES and SR packets over RTCP
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if p.videoSSRC == 0 || p.audioSSRC == 0 {
			continue
		}
		now := time.Now()
		// build sdes packet
		sdes := rtcp.SourceDescription{
			Chunks: []rtcp.SourceDescriptionChunk{{
				Source: uint32(p.videoSSRC),
				Items: []rtcp.SourceDescriptionItem{{
					Type: rtcp.SDESCNAME,
					Text: p.cname,
				}},
			}},
		}

		// build feedback for video ssrc.
		videoPacket := rtcp.CompoundPacket{
			p.videoSenderStream.BuildFeedbackPacket(now, uint32(p.videoSSRC)),
			&sdes,
		}

		// build feedback for audio ssrc.
		audioPacket := rtcp.CompoundPacket{
			p.audioSenderStream.BuildFeedbackPacket(now, uint32(p.audioSSRC)),
			&sdes,
		}

		// send the packets.
		if _, err := p.conn.WriteRTCP(videoPacket); err != nil {
			log.Error().Err(err).Msg("failed to write rtcp")
			return
		}
		if _, err := p.conn.WriteRTCP(audioPacket); err != nil {
			log.Error().Err(err).Msg("failed to write rtcp")
			return
		}
	}
}

func (p *Pipeline) rtcpLoop() {
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
				// log.Info().Msg("Receiver Report")
			case *rtcp.Goodbye:
				log.Info().Msg("Goodbye")
			case *rtcp.TransportLayerNack:
				for _, nack := range pkt.Nacks {
					for _, id := range nack.PacketList() {
						switch webrtc.SSRC(pkt.MediaSSRC) {
						case p.audioSSRC:
							_, q := p.audioSendBuffer.Get(id)
							if q != nil {
								if _, err := p.conn.WriteRTP(q); err != nil {
									log.Error().Err(err).Msg("failed to write rtp")
								}
							}
						case p.videoSSRC:
							_, q := p.videoSendBuffer.Get(id)
							if q != nil {
								if _, err := p.conn.WriteRTP(q); err != nil {
									log.Error().Err(err).Msg("failed to write rtp")
								}
							}
						}
					}
				}
			case *rtcp.TransportLayerCC:
				log.Info().Msg("Transport Layer CC")
			default:
				// log.Info().Msgf("unknown rtcp packet: %v", pkt)
			}
		}
	}
}
