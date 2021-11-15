package rtmp

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

type Conn struct {
	conn io.ReadWriter
}

func Wrap(conn io.ReadWriter) *Conn {
	c := &Conn{
		conn: conn,
	}
	c.start()
	return c
}

func (c *Conn) start() {
	// wait until a C0 packet is received before sending S0 and S1.
	c0 := make([]byte, 1)
	if n, err := c.conn.Read(c0); n != 1 || err != nil {
		log.Warn().Err(err).Msg("failed to read C0 packet")
		return
	}

	// validate c0.
	if c0[0] != 0x03 {
		log.Warn().Msgf("unexpected C0 packet: %x", c0[0])
	}

	// send s0.
	s0 := []byte{0x03}
	if _, err := c.conn.Write(s0); err != nil {
		log.Warn().Err(err).Msg("failed to send S0 packet")
		return
	}

	// send s1.
	s1 := make([]byte, 1536)

	sts0 := uint32(time.Now().Unix())

	binary.BigEndian.PutUint32(s1[0:4], sts0)
	binary.BigEndian.PutUint32(s1[4:8], 0)

	if _, err := rand.Read(s1[8:1536]); err != nil {
		log.Warn().Err(err).Msg("failed to generate random data for S1")
		return
	}

	if _, err := c.conn.Write(s1); err != nil {
		log.Warn().Err(err).Msg("failed to send S1 packet")
		return
	}

	// read c1.
	c1 := make([]byte, 1536)
	if n, err := c.conn.Read(c1); n != 1536 || err != nil {
		log.Warn().Err(err).Msg("failed to read C1 packet")
		return
	}

	cts0 := binary.BigEndian.Uint32(c1[0:4])

	// assert c1[4:8] == 0.
	if c1[4] != 0 || c1[5] != 0 || c1[6] != 0 || c1[7] != 0 {
		log.Warn().Bytes("c1", c1[4:8]).Msg("unexpected C1 packet")
		return
	}

	// send s2.
	s2 := make([]byte, 1536)
	binary.BigEndian.PutUint32(s2[0:4], cts0)

	// write randecho
	copy(s2[8:1536], c1[8:1536])

	if _, err := c.conn.Write(s2); err != nil {
		log.Warn().Err(err).Msg("failed to send C2 packet")
		return
	}

	// read c2.
	c2 := make([]byte, 1536)
	if n, err := c.conn.Read(c2); n != 1536 || err != nil {
		log.Warn().Err(err).Msg("failed to read C2 packet")
		return
	}

	// check eq to sts0
	if binary.BigEndian.Uint32(c2[0:4]) != sts0 {
		log.Warn().Msg("unexpected C2 packet")
		return
	}

	// check next four bytes equal to zero.
	if c2[4] != 0 || c2[5] != 0 || c2[6] != 0 || c2[7] != 0 {
		log.Warn().Msg("unexpected C2 packet")
		return
	}

	// check rand echo.
	if !bytes.Equal(c2[8:1536], s1[8:1536]) {
		log.Warn().Msg("unexpected C2 packet")
		return
	}

	// handshake complete.
}
