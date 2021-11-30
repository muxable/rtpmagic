package main

import (
	"flag"
	"fmt"
	"net"
	"os"
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
	"github.com/rs/zerolog"
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
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	uri := flag.String("uri", "testbin://audio+video", "source uri")
	cname := flag.String("cname", "mugit", "cname to send as")
	netsim := flag.Bool("netsim", false, "use netsim connection")
	debug := flag.String("debug", "", "enable debug rtp destination")
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

	pipeline, err := NewPipeline(conn, *uri, *debug, audioCodec, videoCodec, *cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create pipeline")
	}

	done := make(chan bool)
	go func() {
		for {
			time.Sleep(250 * time.Millisecond)
			targetBitrate := conn.GetEstimatedBitrate()
			// subtract off 64 kbps because it's reserved for audio.
			if targetBitrate < 164_000 {
				// well we're in trouble...
				targetBitrate = 100_000
			}
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

	videoDebugOut *net.UDPConn
	audioDebugOut *net.UDPConn
}

func NewPipeline(conn MuxerUDPConn, uri, debug string, audioCodec, videoCodec *packets.Codec, cname string) (*Pipeline, error) {
	var audioDebugOut *net.UDPConn
	var videoDebugOut *net.UDPConn
	if debug != "" {
		videoAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:5004", debug))
		if err != nil {
			return nil, err
		}
		videoDebugOut, err = net.DialUDP("udp", nil, videoAddr)
		if err != nil {
			return nil, err
		}
		audioAddr, err := net.ResolveUDPAddr("udp",fmt.Sprintf("%s:5006", debug))
		if err != nil {
			return nil, err
		}
		audioDebugOut, err = net.DialUDP("udp", nil, audioAddr)
		if err != nil {
			return nil, err
		}
	}
	p := &Pipeline{
		conn:              conn,
		transcoder:        transcoder.NewTranscoder(uri, audioCodec, videoCodec),
		audioCodec:        audioCodec,
		videoCodec:        videoCodec,
		audioSendBuffer:   nack.NewSendBuffer(12),
		videoSendBuffer:   nack.NewSendBuffer(14),
		audioSenderStream: reports.NewSenderStream(audioCodec.ClockRate),
		videoSenderStream: reports.NewSenderStream(videoCodec.ClockRate),
		cname:             cname,

		videoDebugOut: videoDebugOut,
		audioDebugOut: audioDebugOut,
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
			if p.videoDebugOut != nil {
				buf, err := pkt.Marshal()
				if err != nil {
					log.Error().Err(err).Msg("failed to marshal rtp")
				}
				if _, err := p.videoDebugOut.Write(buf); err != nil {
					log.Error().Err(err).Msg("failed to write video debug")
				}
			}
		case p.audioCodec.PayloadType:
			p.audioSSRC = webrtc.SSRC(pkt.SSRC)
			p.audioSendBuffer.Add(pkt.SequenceNumber, time.Now(), pkt)
			if p.audioDebugOut != nil {
				buf, err := pkt.Marshal()
				if err != nil {
					log.Error().Err(err).Msg("failed to marshal rtp")
				}
				if _, err := p.audioDebugOut.Write(buf); err != nil {
					log.Error().Err(err).Msg("failed to write audio debug")
				}
			}
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
