
package pktque

import (
	"time"
	"github.com/nareix/joy4/av"
)

type Normalizer struct {
	streams []av.CodecData
	time []time.Duration
	MaxAVTimeDiff time.Duration
	timebase time.Duration
}

func (self *Normalizer) Correct(pkt *av.Packet) (ok bool) {
	if len(self.streams) == 0 {
		ok = true
		return
	}

	start, end, correctable, correcttime := self.check(int(pkt.Idx))
	time := pkt.Time - self.timebase
	if time >= start && time < end {
		ok = true
		pkt.Time = time
		self.time[pkt.Idx] = time
	} else {
		if correctable {
			ok = true
			self.timebase = pkt.Time - correcttime
			pkt.Time = correcttime
			self.time[pkt.Idx] = correcttime
		}
	}

	return
}

func (self *Normalizer) check(i int) (start time.Duration, end time.Duration, correctable bool, correcttime time.Duration) {
	minidx := -1
	maxidx := -1
	for j := range self.time {
		if minidx == -1 || self.time[j] < self.time[minidx] {
			minidx = j
		}
		if maxidx == -1 || self.time[j] > self.time[maxidx] {
			maxidx = j
		}
	}
	allthesame := self.time[minidx] == self.time[maxidx]

	if i == minidx {
		if allthesame {
			correctable = true
		} else {
			correctable = false
		}
	} else {
		correctable = true
	}

	start = self.time[minidx]
	end = start + self.MaxAVTimeDiff
	correcttime = start + time.Millisecond*40
	return
}

func NewNormalizer(streams []av.CodecData) *Normalizer {
	n := &Normalizer{}
	n.streams = streams
	n.MaxAVTimeDiff = time.Millisecond*500
	n.time = make([]time.Duration, len(streams))
	return n
}

