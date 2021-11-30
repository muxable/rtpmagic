package rtpnet

import (
	"io"
	"sort"
	"time"

	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/muxer/nack"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtptools/pkg/rfc5761"
	"github.com/muxable/rtptools/pkg/rfc8698"
	"github.com/muxable/rtptools/pkg/rfc8888"
	"github.com/muxable/rtptools/pkg/x_time"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

// CCWrapper wraps an existing connection with a congestion control handler.
type CCWrapper struct {
	conn       io.ReadWriteCloser
	rtpReader  rtpio.RTPReader
	rtcpReader rtpio.RTCPReader
	rtpWriter  rtpio.RTPWriter
	rtcpWriter rtpio.RTCPWriter

	Sender   *rfc8698.Sender
	Receiver *rfc8698.Receiver
	ccBuffer *nack.SendBuffer
	ccSeq    uint16

	Enabled bool

	done chan bool
}

const defaultHdrExtID = 5

func NewCCWrapper(conn io.ReadWriteCloser, mtu int) *CCWrapper {
	rtpReader, rtcpReader := rfc5761.NewReceiver(conn, mtu)
	rtpWriter, rtcpWriter := rfc5761.NewSender(conn)
	now := time.Now()
	done := make(chan bool)
	config := rfc8698.DefaultConfig
	config.MaximumRate = 6 * rfc8698.Mbps
	w := &CCWrapper{
		conn:       conn,
		rtpReader:  rtpReader,
		rtcpReader: rtcpReader,
		rtpWriter:  rtpWriter,
		rtcpWriter: rtcpWriter,
		Sender:     rfc8698.NewSender(now, config),
		Receiver:   rfc8698.NewReceiver(now, config),
		ccBuffer:   nack.NewSendBuffer(14),
		done:       done,
	}

	go func() {
		// periodically poll the receiver and notify the sender.
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		w.sendClockSynchronizationPacket()
		for {
			select {
			case <-ticker.C:
				report := w.Receiver.BuildFeedbackReport()
				w.Sender.OnReceiveFeedbackReport(time.Now(), report)

				w.sendClockSynchronizationPacket()
			case <-done:
				return
			}
		}
	}()

	return w
}

func (w *CCWrapper) sendClockSynchronizationPacket() {
	senderClockPacket, err := packets.NewSenderClockRawPacket(time.Now())
	if err != nil {
		log.Error().Err(err).Msg("failed to create sender clock packet")
		return
	}
	if _, err := w.rtcpWriter.WriteRTCP([]rtcp.Packet{&senderClockPacket}); err != nil {
		log.Error().Err(err).Msg("failed to write sender clock packet")
		return
	}
}

func (w *CCWrapper) ReadRTP(pkt *rtp.Packet) (int, error) {
	return w.rtpReader.ReadRTP(pkt)
}

func (w *CCWrapper) WriteRTP(pkt *rtp.Packet) (int, error) {
	// attach a cc header to this packet.
	tcc, err := (&rtp.TransportCCExtension{TransportSequence: w.ccSeq}).Marshal()
	if err != nil {
		return 0, err
	}
	if err := pkt.Header.SetExtension(defaultHdrExtID, tcc); err != nil {
		return 0, err
	}

	w.ccBuffer.Add(w.ccSeq, time.Now(), pkt)
	w.ccSeq++

	return w.rtpWriter.WriteRTP(pkt)
}

func (w *CCWrapper) ReadRTCP(pkts []rtcp.Packet) (int, error) {
	n, err := w.rtcpReader.ReadRTCP(pkts)
	if err != nil {
		return n, err
	}

	// check if this packet is cc packet.
	for _, pkt := range pkts[:n] {
		switch pkt := pkt.(type) {
		case *rtcp.RawPacket:
			if pkt.Header().Type == rtcp.TypeTransportSpecificFeedback &&
				pkt.Header().Count == rfc8888.FormatCCFB {
				// this is a cc packet.
				report := &rfc8888.RFC8888Report{}
				if err := report.Unmarshal(time.Now(), []byte(*pkt)[8:]); err != nil {
					log.Printf("error %v", err)
					return n, err
				}
				// flatten the report across ssrc's because we send a transport-wide sequence number.
				metrics := []*rfc8888.RFC8888MetricBlock{}

				for _, block := range report.Blocks {
					for _, metric := range block.MetricBlocks {
						if metric == nil {
							continue
						}
						metrics = append(metrics, metric)
					}
				}
				sort.Slice(metrics, func(i, j int) bool {
					return metrics[i].SequenceNumber < metrics[j].SequenceNumber
				})
				for _, metric := range metrics {
					ts, q := w.ccBuffer.Get(metric.SequenceNumber)
					if q == nil {
						log.Warn().Uint16("Seq", metric.SequenceNumber).Msgf("received cc feedback for nonexistent packet")
						continue
					}
					size := rfc8698.Bits(q.MarshalSize() * 8)
					err := w.Receiver.OnReceiveMediaPacket(metric.ArrivalTime, *ts, metric.SequenceNumber, metric.ECN == 0x3, size)
					if err != nil {
						log.Warn().Msgf("cc receiver error: %v", err)
					}
				}
			} else if pkt.Header().Type == rtcp.TypeTransportSpecificFeedback &&
				pkt.Header().Count == 30 {
				report := &packets.ReceiverClock{}
				if err := report.Unmarshal([]byte(*pkt)[4:]); err != nil {
					return n, err
				}
				rtpTime := uint32(x_time.GoTimeToNTP(time.Now()) >> 16)
				delay := time.Duration(float64(report.Delay) / (1 << 16) * float64(time.Second))
				rtt := x_time.NTPToGoDuration(rtpTime-uint32(report.LastSenderNTPTime)) - delay
				w.Enabled = true
				w.Sender.UpdateEstimatedRoundTripTime(rtt)
			}
		}
	}
	return n, err
}

func (w *CCWrapper) WriteRTCP(pkts []rtcp.Packet) (int, error) {
	return w.rtcpWriter.WriteRTCP(pkts)
}

// GetEstimatedBitrate gets the estimated bitrate from the sender.
func (w *CCWrapper) GetEstimatedBitrate() uint32 {
	if !w.Enabled {
		return 100 // send some minimal data.
	}
	return uint32(w.Sender.GetTargetRate(10000))
}

func (w *CCWrapper) Close() error {
	w.done <- true
	return w.conn.Close()
}
