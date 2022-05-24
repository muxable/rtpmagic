package balancer

import (
	"context"
	"io"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/muxable/rtpmagic/api"
	"github.com/muxable/signal/pkg/signal"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/cc"
	"github.com/pion/interceptor/pkg/gcc"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ManagedPeerConnection struct {
	*webrtc.PeerConnection

	connectionState     webrtc.PeerConnectionState
	connectionStateCond *sync.Cond

	bitsTransferred uint64
	lastUpdate      time.Time

	ccs map[string]cc.BandwidthEstimator
}

type ManagedTrack struct {
	tl        *webrtc.TrackLocalStaticRTP
	pc        *ManagedPeerConnection
	source    *ManagedSource
	rtpSender *webrtc.RTPSender
}

type ManagedPeerConnectionGroup struct {
	sync.RWMutex

	addr string

	conns   map[string]*ManagedPeerConnection
	sources map[*ManagedSource]bool

	tracks []*ManagedTrack

	cancel context.CancelFunc
}

type ManagedSource struct {
	codec        webrtc.RTPCodecCapability
	id, streamID string

	mpcg *ManagedPeerConnectionGroup

	readRTCPCh chan []rtcp.Packet

	sendBuffer [1 << 16]*rtp.Packet

	t0 time.Time
}

func NewManagedPeerConnection(addr string, pollingInterval time.Duration) (*ManagedPeerConnectionGroup, error) {
	ctx, cancel := context.WithCancel(context.Background())
	n := &ManagedPeerConnectionGroup{
		addr:    addr,
		conns:   make(map[string]*ManagedPeerConnection),
		sources: make(map[*ManagedSource]bool),
		cancel:  cancel,
	}
	if err := n.bindLocalAddresses(addr); err != nil {
		return nil, err
	}
	go func() {
		ticker := time.NewTicker(pollingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := n.bindLocalAddresses(addr); err != nil {
					log.Warn().Msgf("failed to get local addresses: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return n, nil
}

// bindLocalAddresses binds the local addresses to the UDPConn.
func (n *ManagedPeerConnectionGroup) bindLocalAddresses(addr string) error {
	// get the network interfaces.
	devices, err := GetLocalAddresses()
	if err != nil {
		return err
	}
	n.Lock()
	log.Printf("devices: %v", devices)
	// add any interfaces that are not already active.
	for device := range devices {
		if _, ok := n.conns[device]; !ok {
			if err := n.addDevice(device); err != nil {
				log.Error().Msgf("failed to add device %s: %v", device, err)
				continue
			}

			log.Info().Msgf("connected to %s via %s", addr, device)
		}
	}

	// remove any interfaces that are no longer active.
	for device := range n.conns {
		if _, ok := devices[device]; !ok {
			if err := n.removeDevice(device); err != nil {
				log.Error().Msgf("failed to remove device %s: %v", device, err)
				continue
			}
			log.Info().Msgf("disconnected from %s via %s", addr, device)
		}
	}
	n.Unlock()
	// print some debugging information
	bitrate := n.GetEstimatedBitrate()
	log.Debug().Int("Connections", len(n.conns)).Int("TotalBitrate", bitrate).Msg("active connections")
	keys := make([]string, 0, len(n.conns))
	for key := range n.conns {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		conn := n.conns[key]
		bitrate := conn.GetEstimatedBitrate()
		actual := conn.GetTransferredBitrate()
		log.Debug().Str("Interface", key).Int("TargetBitrate", bitrate).Int("ActualBitrate", actual).Msg("active connection")
	}
	return nil
}

func (mpcg *ManagedPeerConnectionGroup) addDevice(device string) error {
	conn, err := ListenVia(device)
	if err != nil {
		return err
	}

	settingEngine := webrtc.SettingEngine{}

	settingEngine.SetICEUDPMux(webrtc.NewICEUDPMux(nil, conn))

	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeH265,
			ClockRate:    90000,
			RTCPFeedback: []webrtc.RTCPFeedback{{"goog-remb", ""}, {"ccm", "fir"}, {"nack", ""}, {"nack", "pli"}},
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		},
		PayloadType: 97,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	i := &interceptor.Registry{}

	congestionController, err := cc.NewInterceptor(func() (cc.BandwidthEstimator, error) {
		return gcc.NewSendSideBWE(gcc.SendSideBWEInitialBitrate(5_000_000))
	})
	if err != nil {
		return err
	}

	congestionController.OnNewPeerConnection(func(id string, estimator cc.BandwidthEstimator) {
		// add the congestion controller to the managed peer connection.
		mpcg.conns[device].ccs[id] = estimator
	})

	i.Add(congestionController)
	if err := webrtc.ConfigureRTCPReports(i); err != nil {
		return err
	}

	// configure ccnack
	// generator, err := ccnack.NewGeneratorInterceptor()
	// if err != nil {
	// 	return err
	// }

	// responder, err := ccnack.NewResponderInterceptor()
	// if err != nil {
	// 	return err
	// }

	// m.RegisterFeedback(webrtc.RTCPFeedback{Type: "ccnack"}, webrtc.RTPCodecTypeVideo)
	// m.RegisterFeedback(webrtc.RTCPFeedback{Type: "ccnack", Parameter: "pli"}, webrtc.RTPCodecTypeVideo)
	// i.Add(responder)
	// i.Add(generator)

	// this must be after ccnack.
	if err = webrtc.ConfigureTWCCHeaderExtensionSender(m, i); err != nil {
		return err
	}

	mpc := &ManagedPeerConnection{
		ccs:                 make(map[string]cc.BandwidthEstimator),
		connectionState:     webrtc.PeerConnectionStateDisconnected,
		connectionStateCond: sync.NewCond(&sync.Mutex{}),
		lastUpdate:          time.Now(),
	}

	// it's ok if an error is returned since we only dangle a pointer.
	mpcg.conns[device] = mpc

	pc, err := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i), webrtc.WithSettingEngine(settingEngine)).NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		return err
	}

	mpc.PeerConnection = pc

	// create a new signalling channel.
	grpcconn, err := grpc.Dial(mpcg.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client, err := api.NewSFUClient(grpcconn).Publish(context.Background())
	if err != nil {
		return err
	}

	signaller := signal.NewSignaller(pc)

	pc.OnNegotiationNeeded(signaller.Renegotiate)

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		mpc.connectionStateCond.L.Lock()
		mpc.connectionState = state
		mpc.connectionStateCond.Broadcast()
		mpc.connectionStateCond.L.Unlock()
	})

	go func() {
		for {
			pb, err := signaller.ReadSignal()
			if err != nil {
				break
			}
			if err := client.Send(pb); err != nil {
				break
			}
		}
	}()

	go func() {
		for {
			pb, err := client.Recv()
			if err != nil {
				break
			}
			if err := signaller.WriteSignal(pb); err != nil {
				break
			}
		}
	}()

	// add all existing sources.
	for source := range mpcg.sources {
		tl, err := webrtc.NewTrackLocalStaticRTP(source.codec, source.id, source.streamID)
		if err != nil {
			return err
		}
		rtpSender, err := mpc.AddTrack(tl)
		if err != nil {
			return err
		}
		go func() {
			for {
				pkt, _, err := rtpSender.ReadRTCP()
				if err != nil {
					return
				}
				source.readRTCPCh <- pkt
			}
		}()
		mpcg.tracks = append(mpcg.tracks, &ManagedTrack{
			tl:        tl,
			pc:        mpc,
			source:    source,
			rtpSender: rtpSender,
		})
	}

	return nil
}

func (mpcg *ManagedPeerConnectionGroup) removeDevice(device string) error {
	mpcg.Lock()
	defer mpcg.Unlock()

	conn := mpcg.conns[device]

	// remove this interface.
	go conn.Close() // this can block so ignore.
	delete(mpcg.conns, device)

	cleaned := make([]*ManagedTrack, 0, len(mpcg.tracks))
	for _, track := range mpcg.tracks {
		if track.pc == conn {
			if err := track.pc.Close(); err != nil {
				return err
			}
		} else {
			cleaned = append(cleaned, track)
		}
	}

	mpcg.tracks = cleaned

	return nil
}

func (mpcg *ManagedPeerConnectionGroup) AddSource(codec webrtc.RTPCodecCapability, id, streamID string) (*ManagedSource, error) {
	mpcg.Lock()
	defer mpcg.Unlock()

	m := &ManagedSource{readRTCPCh: make(chan []rtcp.Packet), codec: codec, id: id, streamID: streamID, mpcg: mpcg, sendBuffer: [1 << 16]*rtp.Packet{}, t0: time.Now()}

	// add one track for each peer connection in the managed peer connection.
	for _, conn := range mpcg.conns {
		tl, err := webrtc.NewTrackLocalStaticRTP(codec, id, streamID)
		if err != nil {
			return nil, err
		}
		rtpSender, err := conn.AddTrack(tl)
		if err != nil {
			return nil, err
		}
		go func() {
			for {
				pkt, _, err := rtpSender.ReadRTCP()
				if err != nil {
					return
				}
				for _, p := range pkt {
					nack, ok := p.(*rtcp.TransportLayerNack)
					if !ok || nack.SenderSSRC == 0 {
						continue // this is a cc nack.
					}

					for i := range nack.Nacks {
						nack.Nacks[i].Range(func(seq uint16) bool {
							if p := m.sendBuffer[seq]; p != nil {
								log.Printf("resending packet %d", seq)
								if err := m.WriteRTP(p); err != nil {
									log.Error().Err(err).Msg("error sending nack packet")
									return false
								}
							} else {
								log.Warn().Msgf("nack packet not found: %d", seq)
							}
							return true
						})
					}

				}
				m.readRTCPCh <- pkt
			}
		}()
		mpcg.tracks = append(mpcg.tracks, &ManagedTrack{
			tl:        tl,
			pc:        conn,
			source:    m,
			rtpSender: rtpSender,
		})
	}

	mpcg.sources[m] = true

	return m, nil
}

func (pc *ManagedPeerConnectionGroup) RemoveSource(source *ManagedSource) error {
	pc.Lock()
	defer pc.Unlock()

	// for each track in the managed track, remove it from the peer connection.
	cleaned := make([]*ManagedTrack, 0, len(pc.tracks))
	for _, track := range pc.tracks {
		if track.source == source {
			if err := track.pc.RemoveTrack(track.rtpSender); err != nil {
				return err
			}
		} else {
			cleaned = append(cleaned, track)
		}
	}

	pc.tracks = cleaned

	// remove it from the group registry
	delete(pc.sources, source)

	return nil
}

// ReadRTCP reads a single RTCP from the track.
func (m *ManagedSource) ReadRTCP() ([]rtcp.Packet, error) {
	pkt, ok := <-m.readRTCPCh
	if !ok {
		return nil, io.EOF
	}
	return pkt, nil
}

// WriteRTP writes an RTP packet to a random track.
func (m *ManagedSource) WriteRTP(pkt *rtp.Packet) error {
	m.sendBuffer[pkt.SequenceNumber] = pkt.Clone()
	if track := m.randomConn(); track != nil {
		return track.WriteRTP(pkt)
	} else {
		log.Warn().Msg("no track to write to")
	}
	return nil
}

func (t *ManagedTrack) WriteRTP(pkt *rtp.Packet) error {
	t.pc.connectionStateCond.L.Lock()
	for t.pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
		t.pc.connectionStateCond.Wait()
	}
	t.pc.connectionStateCond.L.Unlock()
	t.pc.bitsTransferred += uint64(pkt.MarshalSize() * 8)
	return t.tl.WriteRTP(pkt)
}

func (s *ManagedSource) randomConn() *ManagedTrack {
	bitrates := make(map[*ManagedTrack]int)
	total := 0
	for _, track := range s.mpcg.tracks {
		if track.source != s {
			continue
		}
		bitrates[track] = track.pc.GetEstimatedBitrate()
		total += bitrates[track]
	}
	if total == 0 {
		return nil
	}
	index := rand.Intn(int(total))
	for track, bitrate := range bitrates {
		if index < int(bitrate) {
			return track
		}
		index -= int(bitrate)
	}
	return nil
}

func (pc *ManagedPeerConnection) GetEstimatedBitrate() int {
	if len(pc.ccs) == 0 {
		return 0
	}
	totalBitrate := 0
	for _, cc := range pc.ccs {
		totalBitrate += cc.GetTargetBitrate()
	}
	return totalBitrate / len(pc.ccs)
}

func (pc *ManagedPeerConnection) GetTransferredBitrate() int {
	bitrate := int(float64(pc.bitsTransferred) / time.Since(pc.lastUpdate).Seconds())
	pc.bitsTransferred = 0
	pc.lastUpdate = time.Now()
	return bitrate
}

func (pcg *ManagedPeerConnectionGroup) GetEstimatedBitrate() int {
	totalBitrate := 0
	for _, pc := range pcg.conns {
		totalBitrate += pc.GetEstimatedBitrate()
	}
	return totalBitrate
}

// Close closes all active connections.
func (n *ManagedPeerConnectionGroup) Close() error {
	n.Lock()
	defer n.Unlock()

	n.cancel()
	for _, conn := range n.conns {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}
