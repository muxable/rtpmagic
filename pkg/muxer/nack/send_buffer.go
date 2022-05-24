package nack

import (
	"sync"
	"time"

	"github.com/pion/rtp"
)

const (
	uint16SizeHalf = 1 << 15
)

type SendBuffer struct {
	sync.RWMutex

	packets    []*rtp.Packet
	timestamps []*time.Time
	lastAdded  uint16
	started    bool
}

func NewSendBuffer(size uint16) *SendBuffer {
	return &SendBuffer{
		packets:    make([]*rtp.Packet, 1<<16),
		timestamps: make([]*time.Time, 1<<16),
	}
}

func (s *SendBuffer) Add(seq uint16, ts time.Time, packet *rtp.Packet) {
	s.Lock()
	defer s.Unlock()

	if !s.started {
		s.packets[seq] = packet
		s.timestamps[seq] = &ts
		s.lastAdded = seq
		s.started = true
		return
	}

	diff := seq - s.lastAdded
	if diff == 0 {
		return
	} else if diff < uint16SizeHalf {
		for i := s.lastAdded + 1; i != seq; i++ {
			s.packets[i] = nil
			s.timestamps[i] = nil
		}
	}

	s.packets[seq] = packet
	s.timestamps[seq] = &ts
	s.lastAdded = seq
}

func (s *SendBuffer) Get(seq uint16) (*time.Time, *rtp.Packet) {
	s.RLock()
	defer s.RUnlock()

	diff := s.lastAdded - seq
	if diff >= uint16SizeHalf {
		return nil, nil
	}

	p := s.packets[seq]
	if p.SequenceNumber != seq {
		return nil, nil
	}

	return s.timestamps[seq], p
}
