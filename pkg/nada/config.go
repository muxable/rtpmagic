package nada

import "time"

type BitsPerSecond float64

type Config struct {
	// Weight of priority of the flow
	Priority float64
	// Minimum rate of the application supported by the media encoder
	MinimumRate BitsPerSecond // RMIN
	// Maximum rate of the application supported by media encoder
	MaximumRate BitsPerSecond // RMAX
	// Reference congestion level
	ReferenceCongestionLevel time.Duration // XREF
	// Scaling parameter for gradual rate update calculation
	κ float64
	// Scaling parameter for gradual rate update calculation
	η float64
	// Upper bound of RTT in gradual rate update calculation
	τ time.Duration
	// Target feedback interval
	δ time.Duration

	// Observation window in time for calculating packet summary statistics at receiver
	LogWindow time.Duration // LOGWIN
	// Threshold for determining queuing delay build up at receiver
	QEPS time.Duration
	// Bound on filtering delay
	FilteringDelay time.Duration // DFILT
	// Upper bound on rate increase ratio for accelerated ramp-up
	γ_max float64
	// Upper bound on self-inflicted queueing delay during ramp up
	QueueBound time.Duration // QBOUND

	// Multiplier for self-scaling the expiration threshold of the last observed loss
	// (loss_exp) based on measured average loss interval (loss_int)
	LossMultiplier float64 // MULTILOSS
	// Delay threshold for invoking non-linear warping
	DelayThreshold time.Duration // QTH
	// Scaling parameter in the exponent of non-linear warping
	λ float64

	// Reference packet loss ratio
	ReferencePacketLossRatio float64 // PLRREF
	// Reference packet marking ratio
	ReferencePacketMarkingRatio float64 // PMRREF
	// Reference delay penalty for loss when lacket loss ratio is at least PLRREF
	ReferenceDelayLoss time.Duration // DLOSS
	// Reference delay penalty for ECN marking when packet marking is at PMRREF
	ReferenceDelayMarking time.Duration // DMARK

	// Frame rate of incoming video
	FrameRate float64 // FRAMERATE
	// Scaling parameter for modulating outgoing sending rate
	β_s float64
	// Scaling parameter for modulating video encoder target rate
	β_v float64
	// Smoothing factor in exponential smoothing of packet loss and marking rate
	α float64
}

var DefaultConfig = Config{
	Priority:                 1.0,
	MinimumRate:              BitsPerSecond(150_000),
	MaximumRate:              BitsPerSecond(1_500_000),
	ReferenceCongestionLevel: 10 * time.Millisecond,
	κ:                        0.5,
	η:                        2.0,
	τ:                        500 * time.Millisecond,
	δ:                        100 * time.Millisecond,

	LogWindow:      500 * time.Millisecond,
	QEPS:           10 * time.Millisecond,
	FilteringDelay: 120 * time.Millisecond,
	γ_max:          0.5,
	QueueBound:     50 * time.Millisecond,

	LossMultiplier: 7.0,
	DelayThreshold: 50 * time.Millisecond,
	λ:              0.5,

	ReferencePacketLossRatio:    0.01,
	ReferencePacketMarkingRatio: 0.01,
	ReferenceDelayLoss:          10 * time.Millisecond,
	ReferenceDelayMarking:       2 * time.Millisecond,

	FrameRate: 30.0,
	β_s:       0.1,
	β_v:       0.1,
	α:         0.1,
}
