package pktque

import (
	"github.com/nareix/av"
	"time"
)

type Normalizer struct {
	que []*Queue
	timecr *TimeCorrector
	streams []av.CodecData
	timediff time.Duration
}

func (self *Normalizer) Push(pkt av.Packet) {
	self.timecr.Correct(&pkt)
	i := int(pkt.Idx)
	self.que[i].Push(pkt)
}

func (self *Normalizer) removeBeforeTime(tm time.Duration) {
	for _, que := range self.que {
		for que.Count() > 0 {
			if que.Head().Time < tm {
				que.Pop()
			}
		}
	}
}

func (self *Normalizer) Pop() (pkt av.Packet, dur time.Duration, ok bool) {
	mintm := time.Duration(0)
	minidx := -1
	for i, que := range self.que {
		if que.Count() > 0 {
			if minidx == -1 || que.Head().Time < mintm {
				minidx = i
			}
		}
	}
	if minidx == -1 {
		return
	}

	que := self.que[minidx]
	if que.Count() >= 2 {
		maxgap, defdur := CorrectTimeParams(self.streams[pkt.Idx])
		ok = true
		starttime := que.HeadIdx(0).Time
		endtime := que.HeadIdx(1).Time
		dur = endtime - starttime
		pkt = que.Pop()
		pkt.Time -= self.timediff
		if dur > maxgap {
			dur = defdur
			endtime -= defdur
			self.timediff += endtime - starttime
			self.removeBeforeTime(endtime)
		}
		return
	}

	return
}

func NewNormalizer(streams []av.CodecData) *Normalizer {
	self := &Normalizer{}
	self.que = make([]*Queue, len(streams))
	for i := range self.que {
		self.que[i] = &Queue{}
	}
	self.timecr = NewTimeCorrector(streams)
	self.streams = streams
	return self
}

