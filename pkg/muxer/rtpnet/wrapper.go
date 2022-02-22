package rtpnet

import (
	"net"
	"sort"
	"time"

	"github.com/muxable/rtpmagic/pkg/muxer/nack"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtptools/pkg/rfc8698"
	"github.com/muxable/rtptools/pkg/rfc8888"
	"github.com/muxable/rtptools/pkg/x_time"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

// CCWrapper wraps an existing connection with a congestion control handler.
type CCWrapper struct {
	net.PacketConn

	Sender   *rfc8698.Sender
	Receiver *rfc8698.Receiver
	ccBuffer *nack.SendBuffer
	ccSeq    uint16

	done chan bool
}

const defaultHdrExtID = 5

func NewCCWrapper(conn net.PacketConn) *CCWrapper {
	now := time.Now()
	done := make(chan bool)
	config := rfc8698.DefaultConfig
	// we can tolerate much higher packet losses.
	config.ReferencePacketLossRatio = 0.1
	config.ReferencePacketMarkingRatio = 0.1
	config.MinimumRate = 164 * rfc8698.Kbps
	config.MaximumRate = 4 * rfc8698.Mbps
	w := &CCWrapper{
		PacketConn: conn,
		Sender:     rfc8698.NewSender(now, config),
		Receiver:   rfc8698.NewReceiver(now, config),
		ccBuffer:   nack.NewSendBuffer(14),
		done:       done,
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				report := w.Receiver.BuildFeedbackReport()
				w.Sender.OnReceiveFeedbackReport(time.Now(), report)
			case <-done:
				return
			}
		}
	}()

	return w
}

// func (w *CCWrapper) sendClockSynchronizationPacket() error {
// 	senderClockPacket, err := packets.NewSenderClockRawPacket(time.Now())
// 	if err != nil {
// 		return err
// 	}
// 	buf, err := rtcp.Marshal([]rtcp.Packet{&senderClockPacket})
// 	if err != nil {
// 		return err
// 	}
// 	return w.WriteTo(buf, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 0), Port: 5000})
// 	if err := w.RTCPWriter.WriteRTCP(); err != nil {
// 		return err
// 	}
// 	return nil
// }

// GetEstimatedBitrate gets the estimated bitrate from the sender.
func (w *CCWrapper) GetEstimatedBitrate() (uint32, float64) {
	if w.Sender.SenderEstimatedRoundTripTime == 0 {
		return 1000, 0
	}
	return uint32(w.Sender.GetTargetRate(1000)), w.Receiver.EstimatedPacketLossRatio
}

func (w *CCWrapper) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := w.PacketConn.ReadFrom(b)
	if err != nil {
		return n, addr, err
	}
	// check if this is an rtcp packet.
	pkts, err := rtcp.Unmarshal(b[:n])
	if err != nil {
		return n, addr, nil
	}

	// check if this packet is cc packet.
	for _, pkt := range pkts {
		switch pkt := pkt.(type) {
		case *rtcp.RawPacket:
			if pkt.Header().Type == rtcp.TypeTransportSpecificFeedback &&
				pkt.Header().Count == rfc8888.FormatCCFB {
				// this is a cc packet.
				report := &rfc8888.RFC8888Report{}
				if err := report.Unmarshal(time.Now(), []byte(*pkt)[8:]); err != nil {
					continue
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
					continue
				}
				rtpTime := uint32(x_time.GoTimeToNTP(time.Now()) >> 16)
				delay := time.Duration(float64(report.Delay) / (1 << 16) * float64(time.Second))
				rtt := x_time.NTPToGoDuration(rtpTime-uint32(report.LastSenderNTPTime)) - delay
				w.Sender.UpdateEstimatedRoundTripTime(rtt)
			}
		}
	}
	return n, addr, nil
}

func (w *CCWrapper) WriteTo(b []byte, addr net.Addr) (int, error) {
	pkt := &rtp.Packet{}
	if err := pkt.Unmarshal(b); err != nil {
		return w.PacketConn.WriteTo(b, addr)
	}
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

	c, err := pkt.Marshal()
	if err != nil {
		return 0, err
	}
	return w.PacketConn.WriteTo(c, addr)
}

func (w *CCWrapper) Close() error {
	w.done <- true
	return w.PacketConn.Close()
}
