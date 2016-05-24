package av

import (
	"fmt"
)

type PacketWithIdx struct {
	Idx int
	Packet
}

type Segment struct {
	Pkts []PacketWithIdx
	duration float64
}

func (self Segment) Duration() float64 {
	return self.duration
}

func (self Segment) Concat(seg Segment) (out Segment) {
	out.Pkts = append(self.Pkts, seg.Pkts...)
	out.duration = self.duration+seg.duration
	return
}

func WriteSegment(muxer Muxer, seg Segment) (err error) {
	for _, pkt := range seg.Pkts {
		if err = muxer.WritePacket(pkt.Idx, pkt.Packet); err != nil {
			return
		}
	}
	return
}

type SegmentReader struct {
	Demuxer Demuxer
	streams []CodecData
	vi int
	lastpkt *Packet
}

func (self *SegmentReader) ClearCache() {
	self.lastpkt = nil
}

func (self *SegmentReader) prepare() (err error) {
	self.streams = self.Demuxer.Streams()
	self.vi = -1
	for i, stream := range self.streams {
		if stream.IsVideo() {
			self.vi = i
			break
		}
	}
	if self.vi == -1 {
		err = fmt.Errorf("video stream not found")
		return
	}
	return
}

func (self *SegmentReader) ReadGop() (seg Segment, err error) {
	if len(self.streams) == 0 {
		if err = self.prepare(); err != nil {
			return
		}
	}

	n := 0
	if self.lastpkt != nil {
		n++
		seg.Pkts = append(seg.Pkts, PacketWithIdx{Idx:self.vi, Packet:*self.lastpkt})
		seg.duration += self.lastpkt.Duration
		self.lastpkt = nil
	}

	for {
		var i int
		var pkt Packet
		if i, pkt, err = self.Demuxer.ReadPacket(); err != nil {
			return
		}
		if i == self.vi && pkt.IsKeyFrame {
			n++
		}
		if n == 1 {
			seg.Pkts = append(seg.Pkts, PacketWithIdx{Idx:i, Packet:pkt})
			if i == self.vi {
				seg.duration += pkt.Duration
			}
		} else if n > 1 {
			self.lastpkt = &pkt
			break
		}
	}

	return
}

