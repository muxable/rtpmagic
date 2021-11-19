package main

import (
	"flag"
	"io"
	"net"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/nack"
	"github.com/muxable/rtpmagic/pkg/muxer/transcoder"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/reports"
	"github.com/muxable/rtpmagic/test/netsim"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

func dial(destination string, useNetsim bool) (io.ReadWriteCloser, error) {
	if useNetsim {
		return netsim.NewNetSimUDPConn(destination, []*netsim.ConnectionState{
			{},
		})
	}
	addr, err := net.ResolveUDPAddr("udp", destination)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, addr)
}

func writeRTP(conn io.Writer, p *rtp.Packet) error {
	b, err := p.Marshal()
	if err != nil {
		return err
	}
	return writeBytes(conn, b)
}

func writeRTCP(conn io.Writer, p rtcp.CompoundPacket) error {
	b, err := p.Marshal()
	if err != nil {
		return err
	}
	return writeBytes(conn, b)
}

func writeBytes(conn io.Writer, b []byte) error {
	if _, err := conn.Write(b); err != nil {
		return err
	}
	return nil
}

// the general pipeline is
// audio/video src as string -> raw
// encode audio and video
// broadcast rtp
func main() {
	rtmp := flag.String("uri", "testbin://audio+video", "source uri")
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

	pipeline, err := NewPipeline(conn, *rtmp, audioCodec, videoCodec, *cname)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create pipeline")
	}
	
	pipeline.Start()
}

type Pipeline struct {
	sync.RWMutex
	conn              io.ReadWriteCloser
	transcoder        *transcoder.Transcoder
	audioSSRC         uint32
	videoSSRC         uint32
	audioCodec        *packets.Codec
	videoCodec        *packets.Codec
	audioSendBuffer   *nack.SendBuffer
	videoSendBuffer   *nack.SendBuffer
	audioSenderStream *reports.SenderStream
	videoSenderStream *reports.SenderStream
	cname             string
}

func NewPipeline(conn io.ReadWriteCloser, uri string, audioCodec *packets.Codec, videoCodec *packets.Codec, cname string) (*Pipeline, error) {
	videoSendBuffer, err := nack.NewSendBuffer(1 << 14)
	if err != nil {
		return nil, err
	}
	audioSendBuffer, err := nack.NewSendBuffer(1 << 12)
	if err != nil {
		return nil, err
	}
	
	p := &Pipeline{
		conn:              conn,
		transcoder:        transcoder.NewTranscoder(uri, audioCodec.MimeType, videoCodec.MimeType),
		audioCodec:        audioCodec,
		videoCodec:        videoCodec,
		audioSendBuffer:   audioSendBuffer,
		videoSendBuffer:   videoSendBuffer,
		audioSenderStream: reports.NewSenderStream(audioCodec.ClockRate),
		videoSenderStream: reports.NewSenderStream(videoCodec.ClockRate),
		cname:             cname,
	}

	go p.writeRTCPLoop()
	go p.rtcpLoop()

	return p, nil
}

func (p *Pipeline) Start() {
	for {
		pkt := &rtp.Packet{}
		_, err := p.transcoder.ReadRTP(pkt)
		if err != nil {
			log.Warn().Err(err).Msg("failed to read rtp packet")
			continue
		}
		switch pkt.PayloadType {
		case p.videoCodec.PayloadType:
			p.videoSSRC = pkt.SSRC
			p.videoSendBuffer.Add(pkt)
		case p.audioCodec.PayloadType:
			p.audioSSRC = pkt.SSRC
			p.audioSendBuffer.Add(pkt)
		}
		if err := writeRTP(p.conn, pkt); err != nil {
			log.Error().Err(err).Msg("failed to write rtp")
		}
	}
}

func (p *Pipeline) matchesSSRCs(ssrcs []uint32) bool {
	for _, ssrc := range ssrcs {
		if p.audioSSRC == ssrc || p.videoSSRC == ssrc {
			return true
		}
	}
	return false
}

func (p *Pipeline) writeRTCPLoop() {
	// send forward SDES and SR packets over RTCP
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if p.videoSSRC == 0 || p.audioSSRC == 0 {
				continue
			}
			now := time.Now()
			// build sdes packet
			sdes := rtcp.SourceDescription{
				Chunks: []rtcp.SourceDescriptionChunk{{
					Source: p.videoSSRC,
					Items: []rtcp.SourceDescriptionItem{{
						Type: rtcp.SDESCNAME,
						Text: p.cname,
					}},
				}},
			}

			// build feedback for video ssrc.
			videoPacket := rtcp.CompoundPacket{
				p.videoSenderStream.BuildFeedbackPacket(now, p.videoSSRC),
				&sdes,
			}

			// build feedback for audio ssrc.
			audioPacket := rtcp.CompoundPacket{
				p.audioSenderStream.BuildFeedbackPacket(now, p.audioSSRC),
				&sdes,
			}

			// send the packets.
			if err := writeRTCP(p.conn, videoPacket); err != nil {
				log.Error().Err(err).Msg("failed to write rtcp")
			}
			if err := writeRTCP(p.conn, audioPacket); err != nil {
				log.Error().Err(err).Msg("failed to write rtcp")
			}
		}
	}
}

func (p *Pipeline) rtcpLoop() {
	for {
		buf := make([]byte, 1500)
		n, err := p.conn.Read(buf)
		if err != nil {
			log.Warn().Err(err).Msg("connection error")
			return
		}
		// assume these are all rtcp packets.
		cp, err := rtcp.Unmarshal(buf[:n])
		if err != nil {
			log.Warn().Err(err).Msg("rtcp unmarshal error")
			continue
		}
		for _, pkt := range cp {
			// we might get packets for unrelated SSRCs, so discard this packet if it's not relevant to us.
			if !p.matchesSSRCs(pkt.DestinationSSRC()) {
				log.Warn().Msg("discarding packet for unrelated SSRC")
				continue
			}
			switch pkt := pkt.(type) {
			case *rtcp.PictureLossIndication:
				log.Info().Msg("PLI")
			case *rtcp.ReceiverReport:
				log.Info().Msg("Receiver Report")
			case *rtcp.Goodbye:
				log.Info().Msg("Goodbye")
			case *rtcp.TransportLayerNack:
				for _, nack := range pkt.Nacks {
					for _, id := range nack.PacketList() {
						switch pkt.MediaSSRC {
						case p.audioSSRC:
							q := p.audioSendBuffer.Get(id)
							if q != nil {
								if err := writeRTP(p.conn, q); err != nil {
									log.Error().Err(err).Msg("failed to write rtp")
								}
							}
						case p.videoSSRC:
							q := p.videoSendBuffer.Get(id)
							if q != nil {
								if err := writeRTP(p.conn, q); err != nil {
									log.Error().Err(err).Msg("failed to write rtp")
								}
							}
						}
					}
				}
			case *rtcp.TransportLayerCC:
				log.Info().Msg("Transport Layer CC")
			default:
				log.Warn().Interface("Packet", pkt).Msg("unknown rtcp packet")
			}
		}
	}
}
