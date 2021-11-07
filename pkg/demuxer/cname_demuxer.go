package demuxer

import (
	"time"

	"github.com/muxable/rtpmagic/pkg/pipeline"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/rs/zerolog/log"
)

type CNAMEDemuxer struct {
	ctx    pipeline.Context
	rtpIn  chan *rtp.Packet
	rtcpIn chan *rtcp.Packet
	out    chan CNAMESource

	bySSRC  map[uint32]*CNAMESource
	byCNAME map[string]*CNAMESource
}

type CNAMESource struct {
	CNAME string
	RTP   chan *rtp.Packet
	RTCP  chan *rtcp.Packet

	lastPacket time.Time
}

// NewCNAMEDemuxer creates a new CNAMEDemuxer
func NewCNAMEDemuxer(ctx pipeline.Context, rtpIn chan *rtp.Packet, rtcpIn chan *rtcp.Packet) chan CNAMESource {
	out := make(chan CNAMESource)
	d := &CNAMEDemuxer{
		ctx:     ctx,
		rtpIn:   rtpIn,
		rtcpIn:  rtcpIn,
		out:     out,
		bySSRC:  make(map[uint32]*CNAMESource),
		byCNAME: make(map[string]*CNAMESource),
	}
	go d.start()
	return out
}

// start starts the CNAMEDemuxer
func (d *CNAMEDemuxer) start() {
	// if debug mode is on, add a default cname to make testing easier.
	s := &CNAMESource{
		CNAME: "mugit",
		RTP:   make(chan *rtp.Packet),
		RTCP:  make(chan *rtcp.Packet),
	}
	d.out <- *s
	d.byCNAME["mugit"] = s

	for {
		select {
		case p, ok := <-d.rtpIn:
			if !ok {
				return
			}
			d.handleRTP(p)
		case p, ok := <-d.rtcpIn:
			if !ok {
				return
			}
			d.handleRTCP(p)
		case <-d.ctx.Clock.After(time.Second):
			d.cleanup()
		}
	}
}

// handleRTP checks if the RTP's SSRC is registered and if so, forwards it on to that source
func (d *CNAMEDemuxer) handleRTP(p *rtp.Packet) {
	if s, ok := d.bySSRC[p.SSRC]; ok {
		s.RTP <- p
	} else {
		// log.Warn().Uint32("SSRC", p.SSRC).Msg("ssrc received with unknown cname")
		d.byCNAME["mugit"].RTP <- p
	}
}

// handleRTCP registers a given SSRC to a CNAMESource.
func (d *CNAMEDemuxer) handleRTCP(p *rtcp.Packet) {
	switch p := (*p).(type) {
	case *rtcp.SourceDescription:
		for _, c := range p.Chunks {
			ssrc := c.Source
			for _, item := range c.Items {
				if item.Type == rtcp.SDESCNAME {
					cname := item.Text

					s, ok := d.byCNAME[cname]
					if !ok {
						// create a new source
						s = &CNAMESource{
							CNAME: cname,
							RTP:   make(chan *rtp.Packet),
							RTCP:  make(chan *rtcp.Packet),
						}

						// notify the pipeline that a new source has been created
						d.out <- *s

						// log a new source
						log.Info().Str("CNAME", cname).Msg("new cname")
					}
					// update the source's last packet time
					s.lastPacket = d.ctx.Clock.Now()

					// make sure the source is registered to the SSRC
					if d.bySSRC[ssrc] != s {
						// log the addition
						log.Info().Str("CNAME", cname).Uint32("SSRC", ssrc).Msg("adding ssrc to cname")
					}
					d.bySSRC[ssrc] = s
				}
			}
		}
	}
}

// cleanup removes any cname sources that haven't received a packet in the last 30 seconds
func (d *CNAMEDemuxer) cleanup() {
	now := d.ctx.Clock.Now()
	for cname, s := range d.byCNAME {
		if now.Sub(s.lastPacket) > 30*time.Second && cname != "mugit" {
			// log the removal
			log.Info().Str("CNAME", cname).Msg("removing cname due to timeout")
			delete(d.byCNAME, cname)
			// go through all the ssrc's and delete the ones that match this source.
			for ssrc, t := range d.bySSRC {
				if s == t {
					delete(d.bySSRC, ssrc)
				}
			}
			// close the output channels
			close(s.RTP)
			close(s.RTCP)
		}
	}
}
