package nada

import (
	"math"
	"time"
)

type Sender struct {
	config                            Config
	ReferenceRate                     BitsPerSecond // r_ref
	SenderEstimatedRoundTripTime      time.Duration // rtt
	PreviousAggregateCongestionSignal time.Duration // x_prev
	LastTimestamp                     time.Time
	CurrentTimestamp                  time.Time
}

func NewSender(now time.Time, config Config) *Sender {
	return &Sender{
		config:                            config,
		ReferenceRate:                     config.MinimumRate,
		SenderEstimatedRoundTripTime:      0,
		PreviousAggregateCongestionSignal: 0,
		LastTimestamp:                     now,
		CurrentTimestamp:                  now,
	}
}

func (s *Sender) OnReceiveFeedbackReport(now time.Time, report NADAReport) {
	// obtain current timestamp from system clock: t_curr
	s.CurrentTimestamp = now

	// update estimation of rtt
	// TODO: implement

	// measure feedback interval: delta = t_curr - t_last
	delta := s.CurrentTimestamp.Sub(s.LastTimestamp)

	if report.RecommendedRateAdaptionMode {
		// update r_ref following gradual update rules
		//
		// In gradual update mode, the rate r_ref is updated as:
		//
		//    x_offset = x_curr - PRIO*XREF*RMAX/r_ref          (5)
		//
		//    x_diff   = x_curr - x_prev                        (6)
		//
		//                           delta    x_offset
		//    r_ref = r_ref - KAPPA*-------*------------*r_ref
		//                            TAU       TAU
		//
		//                                x_diff
		//                  - KAPPA*ETA*---------*r_ref         (7)
		//                                 TAU

		x_offset := report.AggregatedCongestionSignal - scale(s.config.ReferenceCongestionLevel, s.config.Priority*float64(s.config.MaximumRate)/float64(s.ReferenceRate))
		x_diff := report.AggregatedCongestionSignal - s.PreviousAggregateCongestionSignal

		s.ReferenceRate = BitsPerSecond(float64(s.ReferenceRate) *
			(1 -
				(s.config.κ * (float64(delta) / float64(s.config.τ)) * (float64(x_offset) / float64(s.config.τ))) -
				(s.config.κ * s.config.η * (float64(x_diff) / float64(s.config.τ)))))
	} else {
		// update r_ref following accelerated ramp-up rules
		//
		// In accelerated ramp-up mode, the rate r_ref is updated as follows:
		//
		//                                    QBOUND
		//        gamma = min(GAMMA_MAX, ------------------)     (3)
		//                                rtt+DELTA+DFILT
		//
		//        r_ref = max(r_ref, (1+gamma) r_recv)           (4)

		γ := math.Min(s.config.γ_max, float64(s.config.QueueBound)/float64(s.SenderEstimatedRoundTripTime+s.config.δ+s.config.FilteringDelay))
		s.ReferenceRate = BitsPerSecond(math.Max(float64(s.ReferenceRate), (1+γ)*float64(report.ReceivingRate)))
	}

	// clip rate r_ref within the range of minimum rate (RMIN) and maximum rate (RMAX).
	if s.ReferenceRate < s.config.MinimumRate {
		s.ReferenceRate = s.config.MinimumRate
	}
	if s.ReferenceRate > s.config.MaximumRate {
		s.ReferenceRate = s.config.MaximumRate
	}

	// x_prev = x_curr
	s.PreviousAggregateCongestionSignal = report.AggregatedCongestionSignal

	// t_last = t_curr
	s.LastTimestamp = s.CurrentTimestamp
}

func (s *Sender) GetTargetRate(bufferLen uint) BitsPerSecond {
	// r_diff_v = min(0.05*r_ref, BETA_V*8*buffer_len*FPS).     (11)
	// r_vin  = max(RMIN, r_ref - r_diff_v).      (13)

	r_diff_v := math.Min(0.05*float64(s.ReferenceRate), s.config.β_v*8*float64(bufferLen)*float64(s.config.FrameRate))
	r_vin := math.Max(float64(s.config.MinimumRate), float64(s.ReferenceRate)-r_diff_v)
	return BitsPerSecond(r_vin)
}

func (s *Sender) GetSendingRate(bufferLen uint) BitsPerSecond {
	// r_diff_s = min(0.05*r_ref, BETA_S*8*buffer_len*FPS).     (12)
	// r_send = min(RMAX, r_ref + r_diff_s).    (14)

	r_diff_s := math.Min(0.05*float64(s.ReferenceRate), s.config.β_s*8*float64(bufferLen)*float64(s.config.FrameRate))
	r_send := math.Min(float64(s.config.MaximumRate), float64(s.ReferenceRate)+r_diff_s)
	return BitsPerSecond(r_send)
}
