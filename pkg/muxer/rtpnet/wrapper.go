package rtpnet

import (
	"io"
	"sort"
	"time"

	"github.com/muxable/rtpio"
	"github.com/muxable/rtpmagic/pkg/muxer/nack"
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

	ccSender   *rfc8698.Sender
	ccReceiver *rfc8698.Receiver
	ccBuffer   *nack.SendBuffer
	ccSeq      uint16

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
		ccSender:   rfc8698.NewSender(now, config),
		ccReceiver: rfc8698.NewReceiver(now, config),
		ccBuffer:   nack.NewSendBuffer(14),
		done:       done,
	}

	go func() {
		// periodically poll the receiver and notify the sender.
		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				report := w.ccReceiver.BuildFeedbackReport()
				w.ccSender.OnReceiveFeedbackReport(time.Now(), report)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return w
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
	for _, pkt := range pkts {
		switch pkt := pkt.(type) {
		case *rtcp.ReceiverReport:
			rtpTime := uint32(x_time.GoTimeToNTP(time.Now()) >> 16)
			for _, report := range pkt.Reports {
				// since rtt is ssrc independent it doesn't matter which report we use.
				if report.LastSenderReport == 0 {
					continue
				}
				delay := time.Duration(float64(report.Delay)/(1<<16) * float64(time.Second))
				rtt := x_time.NTPToGoDuration(rtpTime-report.LastSenderReport) - delay
				w.ccSender.UpdateEstimatedRoundTripTime(rtt)
				break
			}
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
					err := w.ccReceiver.OnReceiveMediaPacket(metric.ArrivalTime, *ts, metric.SequenceNumber, metric.ECN == 0x3, size)
					if err != nil {
						log.Warn().Msgf("cc receiver error: %v", err)
					}
				}
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
	return uint32(w.ccSender.GetTargetRate(10000))
}

func (w *CCWrapper) Close() error {
	w.done <- true
	return w.conn.Close()
}