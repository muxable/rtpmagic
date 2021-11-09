package nack

import (
	"fmt"
	"sync"

	"github.com/pion/rtp"
)

const (
	uint16SizeHalf = 1 << 15
)

type SendBuffer struct {
	packets   []*rtp.Packet
	size      uint16
	lastAdded uint16
	started   bool

	m sync.RWMutex
}

func NewSendBuffer(size uint16) (*SendBuffer, error) {
	allowedSizes := make([]uint16, 0)
	correctSize := false
	for i := 0; i < 16; i++ {
		if size == 1<<i {
			correctSize = true
			break
		}
		allowedSizes = append(allowedSizes, 1<<i)
	}

	if !correctSize {
		return nil, fmt.Errorf("%w: %d is not a valid size, allowed sizes: %v", ErrInvalidSize, size, allowedSizes)
	}

	return &SendBuffer{
		packets: make([]*rtp.Packet, size),
		size:    size,
	}, nil
}

func (s *SendBuffer) Add(packet *rtp.Packet) {
	s.m.Lock()
	defer s.m.Unlock()

	seq := packet.SequenceNumber
	if !s.started {
		s.packets[seq%s.size] = packet
		s.lastAdded = seq
		s.started = true
		return
	}

	diff := seq - s.lastAdded
	if diff == 0 {
		return
	} else if diff < uint16SizeHalf {
		for i := s.lastAdded + 1; i != seq; i++ {
			s.packets[i%s.size] = nil
		}
	}

	s.packets[seq%s.size] = packet
	s.lastAdded = seq
}

func (s *SendBuffer) Get(seq uint16) *rtp.Packet {
	s.m.RLock()
	defer s.m.RUnlock()

	diff := s.lastAdded - seq
	if diff >= uint16SizeHalf {
		return nil
	}

	if diff >= s.size {
		return nil
	}

	return s.packets[seq%s.size]
}