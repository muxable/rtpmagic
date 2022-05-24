module github.com/muxable/rtpmagic

go 1.17

replace github.com/muxable/bluetooth => ../bttest

replace github.com/muxable/sfu => ../sfu

require (
	github.com/google/uuid v1.3.0
	github.com/muxable/sfu v0.0.0-20220519231102-5b3505cc2042
	github.com/muxable/signal v0.0.0-20220312145144-4c0e0ca92a2c
	github.com/pion/interceptor v0.1.10
	github.com/pion/rtcp v1.2.9
	github.com/pion/rtp v1.7.9
	github.com/pion/rtpio v0.1.4
	github.com/pion/webrtc/v3 v3.1.23
	github.com/rs/zerolog v1.26.1
	go.uber.org/zap v1.21.0
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.27.1
)

require golang.org/x/net v0.0.0-20220401154927-543a649e0bdd // indirect

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/pion/datachannel v1.5.2 // indirect
	github.com/pion/dtls/v2 v2.1.3 // indirect
	github.com/pion/ice/v2 v2.2.2 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.5 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/sctp v1.8.2 // indirect
	github.com/pion/sdp v1.3.0 // indirect
	github.com/pion/sdp/v3 v3.0.4 // indirect
	github.com/pion/srtp/v2 v2.0.5 // indirect
	github.com/pion/stun v0.3.5 // indirect
	github.com/pion/transport v0.13.0 // indirect
	github.com/pion/turn/v2 v2.0.8 // indirect
	github.com/pion/udp v0.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/goleak v1.1.12 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20220131195533-30dcbda58838 // indirect
	golang.org/x/sys v0.0.0-20220128215802-99c3d69c2c27 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20220126215142-9970aeb2e350 // indirect
)
