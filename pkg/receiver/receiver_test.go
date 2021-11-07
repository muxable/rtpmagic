package receiver

import (
	"net"
	"reflect"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/muxable/rtpmagic/pkg/packets"
	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"go.uber.org/goleak"
)

func TestReceiver_Simple(t *testing.T) {
	ctx := pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{PayloadType: 0},
		}),
		Clock: clock.New(),
	}
	rtcpReturn := make(chan rtcp.Packet)
	rtpOut, _, err := NewReceiver(ctx, "0.0.0.0:5738", rtcpReturn)
	if err != nil {
		t.Fatal(err)
	}

	// send some rtp packets.
	p1 := rtp.Packet{Header: rtp.Header{SequenceNumber: 1}}
	p2 := rtp.Packet{Header: rtp.Header{SequenceNumber: 2}}
	p3 := rtp.Packet{Header: rtp.Header{SequenceNumber: 3}}

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5738})
	if err != nil {
		t.Fatal(err)
	}

	b1, err := p1.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	b2, err := p2.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	b3, err := p3.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Write(b1); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(b2); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(b3); err != nil {
		t.Fatal(err)
	}

	// check that the packets are received
	v1 := <-rtpOut
	if v1.Header.SequenceNumber != 1 {
		t.Errorf("got %d, want 1", v1.Header.SequenceNumber)
	}
	v2 := <-rtpOut
	if v2.Header.SequenceNumber != 2 {
		t.Errorf("got %d, want 2", v2.Header.SequenceNumber)
	}
	v3 := <-rtpOut
	if v3.Header.SequenceNumber != 3 {
		t.Errorf("got %d, want 3", v3.Header.SequenceNumber)
	}

	close(rtcpReturn)

	conn.Close()

	goleak.VerifyNone(t)
}

func TestReceiver_MultipleConnections(t *testing.T) {
	ctx := pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{PayloadType: 0},
		}),
		Clock: clock.New(),
	}
	rtcpReturn := make(chan rtcp.Packet)
	rtpOut, _, err := NewReceiver(ctx, "0.0.0.0:5738", rtcpReturn)
	if err != nil {
		t.Fatal(err)
	}

	// send some rtp packets.
	p1 := rtp.Packet{Header: rtp.Header{SequenceNumber: 1}}
	p2 := rtp.Packet{Header: rtp.Header{SequenceNumber: 2}}
	p3 := rtp.Packet{Header: rtp.Header{SequenceNumber: 3}}

	conn1, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5738})
	if err != nil {
		t.Fatal(err)
	}
	conn2, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5738})
	if err != nil {
		t.Fatal(err)
	}

	b1, err := p1.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	b2, err := p2.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	b3, err := p3.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn1.Write(b1); err != nil {
		t.Fatal(err)
	}
	if _, err := conn2.Write(b2); err != nil {
		t.Fatal(err)
	}
	conn2.Close()
	if _, err := conn1.Write(b3); err != nil {
		t.Fatal(err)
	}

	// check that the packets are received
	v1 := <-rtpOut
	if v1.Header.SequenceNumber != 1 {
		t.Errorf("got %d, want 1", v1.Header.SequenceNumber)
	}
	v2 := <-rtpOut
	if v2.Header.SequenceNumber != 2 {
		t.Errorf("got %d, want 2", v2.Header.SequenceNumber)
	}
	v3 := <-rtpOut
	if v3.Header.SequenceNumber != 3 {
		t.Errorf("got %d, want 3", v3.Header.SequenceNumber)
	}

	close(rtcpReturn)

	conn1.Close()

	goleak.VerifyNone(t)
}

func TestReceiver_WritesRTCP(t *testing.T) {
	ctx := pipeline.Context{
		Codecs: packets.NewCodecSet([]packets.Codec{
			{PayloadType: 0},
		}),
		Clock: clock.New(),
	}
	rtcpReturn := make(chan rtcp.Packet, 10)
	rtpOut, _, err := NewReceiver(ctx, "0.0.0.0:5738", rtcpReturn)
	if err != nil {
		t.Fatal(err)
	}

	// send some rtp packets.
	p1 := rtp.Packet{Header: rtp.Header{SequenceNumber: 1, SSRC: 0xbc5e9a40}}

	rtcp1 := &rtcp.ReceiverReport{
		SSRC: 0x902f9e2e,
		Reports: []rtcp.ReceptionReport{{
			SSRC:               0xbc5e9a40,
			FractionLost:       0,
			TotalLost:          0,
			LastSequenceNumber: 0x46e1,
			Jitter:             273,
			LastSenderReport:   0x9f36432,
			Delay:              150137,
		}},
		ProfileExtensions: []byte{},
	}

	conn1, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5738})
	if err != nil {
		t.Fatal(err)
	}

	b1, err := p1.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	b2, err := rtcp1.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn1.Write(b1); err != nil {
		t.Fatal(err)
	}

	// check the rtp packet was received.
	v1 := <-rtpOut
	if v1.Header.SequenceNumber != 1 {
		t.Errorf("got %d, want 1", v1.Header.SequenceNumber)
	}

	// send the rtcp packet
	rtcpReturn <- rtcp1

	// check the rtcp packet was received.
	b3 := make([]byte, 1500)
	n, err := conn1.Read(b3)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(b3[:n], b2) {
		t.Errorf("got %v, want %v", b3[:n], b2)
	}

	close(rtcpReturn)

	conn1.Close()

	goleak.VerifyNone(t)
}
