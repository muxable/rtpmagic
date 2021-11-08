package main

import (
	"net"

	"github.com/muxable/rtpmagic/test"
	"github.com/rs/zerolog/log"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:5001")
	if err != nil {
		panic(err)
	}
	in, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	out, err := test.NewNetSimUDPConn("127.0.0.1:5000", []*test.SimulatedConnection{
		{
			DropRate: 0.10,
		},
	})
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := out.Read(buf)
			if err != nil {
				return
			}
			if _, err := in.Write(buf[:n]); err != nil {
				log.Warn().Err(err).Msg("failed to write")
				return
			}
		}
	}()

	for {
		buf := make([]byte, 1500)
		n, err := in.Read(buf)
		if err != nil {
			panic(err)
		}
		out.Write(buf[:n])
	}
}