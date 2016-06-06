package pktqueue

import (
	"github.com/nareix/av"
)

type timePacket struct {
	time float64
	av.Packet
}

type stream struct {
	pkts []timePacket
	lastNormalDuration float64
	isVideo bool
}

func (self *stream) isDurationNormal(v float64) bool {
	return v > float64(0.0) && v < float64(1.0)
}

func (self *stream) Read(flush bool) (ok bool, pkt timePacket) {
	if len(self.pkts) > 1 {
		ok = true
		pkt = self.pkts[0]
		pkt.Duration = self.pkts[1].time - self.pkts[0].time
		self.pkts = self.pkts[1:]
	} else if len(self.pkts) == 1 && flush {
		ok = true
		pkt = self.pkts[0]
		pkt.Duration = self.lastNormalDuration
		self.pkts = self.pkts[1:]
	}
	if ok {
		if !self.isDurationNormal(pkt.Duration) {
			pkt.Duration = self.lastNormalDuration
		} else {
			self.lastNormalDuration = pkt.Duration
		}
	}
	return
}

type Queue struct {
	streams []*stream
	Poll func() error
	err error
	time float64
}

func (self *Queue) CurrentTime() float64 {
	return self.time
}

func (self *Queue) Alloc(streams []av.CodecData) {
	self.streams = make([]*stream, len(streams))
	for i := 0; i < len(self.streams); i++ {
		self.streams[i] = &stream{
			isVideo: streams[i].IsVideo(),
		}
	}
}

func (self *Queue) Clear() {
	for _, stream := range self.streams {
		stream.pkts = []timePacket{}
		stream.lastNormalDuration = float64(0.0)
	}
	self.time = 0.0
}

func (self *Queue) ReadPacket() (i int, pkt av.Packet, err error) {
	for {
		flush := self.err != nil
		var tpkt timePacket
		var ok bool
		var stream *stream
		for i, stream = range self.streams {
			if ok, tpkt = stream.Read(flush); ok {
				break
			}
		}
		if ok {
			pkt = tpkt.Packet
			self.time = tpkt.time
			return
		} else {
			if self.err == nil {
				self.err = self.Poll()
			} else {
				err = self.err
				return
			}
		}
	}
	return
}

func (self *Queue) WriteTimePacket(i int, time float64, pkt av.Packet) {
	stream := self.streams[i]
	stream.pkts = append(stream.pkts, timePacket{Packet: pkt, time: time})
}

