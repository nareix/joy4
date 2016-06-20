package pktque

import (
	"github.com/nareix/av"
	"time"
)

type TimeCorrector struct {
	streams []av.CodecData
	intimes []time.Duration
	indurs []time.Duration
	intime time.Duration
	outtime time.Duration
}

func NewTimeCorrector(streams []av.CodecData) *TimeCorrector {
	self := &TimeCorrector{}
	self.intimes = make([]time.Duration, len(streams))
	self.indurs = make([]time.Duration, len(streams))
	self.streams = streams
	return self
}

func (self *TimeCorrector) updateIntimes() {
	for i := range self.intimes {
		self.intimes[i] = self.intime
	}
}

func (self *TimeCorrector) Correct(pkt *av.Packet) {
	i := int(pkt.Idx)

	if pkt.Time < self.intime {
		self.intime = pkt.Time
		self.updateIntimes()
	} else {
		diff := pkt.Time - self.intimes[i]
		maxgap, defdur := CorrectTimeParams(self.streams[i])

		if diff > maxgap {
			var outdiff time.Duration
			dur := self.indurs[i]
			if dur == time.Duration(0) {
				dur = defdur
			}
			adjust := self.intimes[i]+dur
			if adjust > self.intime {
				outdiff = adjust - self.intime
			}
			self.outtime += outdiff
			self.intime = pkt.Time
			self.updateIntimes()
		} else {
			self.indurs[i] = pkt.Time-self.intimes[i]
			self.intimes[i] = pkt.Time
			self.outtime += pkt.Time-self.intime
			self.intime = pkt.Time
		}
	}

	pkt.Time = self.outtime
	return
}

func CorrectTimeParams(stream av.CodecData) (maxgap time.Duration, dur time.Duration) {
	if stream.Type().IsAudio() {
		maxgap = time.Millisecond*500
		dur = time.Millisecond*20
	} else {
		maxgap = time.Millisecond*500
		dur = time.Millisecond*20
	}
	return
}

