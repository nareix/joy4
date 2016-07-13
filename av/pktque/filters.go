
package pktque

import (
	"time"
	"github.com/nareix/joy4/av"
)

type Filter interface {
	ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error)
}

type Filters []Filter

func (self Filters) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	for _, filter := range self {
		if drop, err = filter.ModifyPacket(pkt, streams, videoidx, audioidx); err != nil {
			return
		}
		if drop {
			return
		}
	}
	return
}

type FilterDemuxer struct {
	av.Demuxer
	Filter Filter
	streams []av.CodecData
	videoidx int
	audioidx int
}

func (self FilterDemuxer) ReadPacket() (pkt av.Packet, err error) {
	if self.streams == nil {
		if self.streams, err = self.Demuxer.Streams(); err != nil {
			return
		}
		for i, stream := range self.streams {
			if stream.Type().IsVideo() {
				self.videoidx = i
			} else if stream.Type().IsAudio() {
				self.audioidx = i
			}
		}
	}

	for {
		if pkt, err = self.Demuxer.ReadPacket(); err != nil {
			return
		}
		var drop bool
		if drop, err = self.Filter.ModifyPacket(&pkt, self.streams, self.videoidx, self.audioidx); err != nil {
			return
		}
		if !drop {
			break
		}
	}

	return
}

type WaitKeyFrame struct {
	ok bool
}

func (self *WaitKeyFrame) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	if !self.ok && pkt.Idx == int8(videoidx) && pkt.IsKeyFrame {
		self.ok = true
	}
	drop = !self.ok
	return
}

type TimeStartFromZero struct {
	timebase time.Duration
}

func (self *TimeStartFromZero) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	if self.timebase == 0 {
		self.timebase = pkt.Time
	}
	pkt.Time -= self.timebase
	return
}

type AVSync struct {
	MaxTimeDiff time.Duration
	time []time.Duration
	timebase time.Duration
}

func (self *AVSync) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	if self.time == nil {
		self.time = make([]time.Duration, len(streams))
		if self.MaxTimeDiff == 0 {
			self.MaxTimeDiff = time.Millisecond*500
		}
	}

	start, end, correctable, correcttime := self.check(int(pkt.Idx))
	time := pkt.Time - self.timebase
	if time >= start && time < end {
		pkt.Time = time
		self.time[pkt.Idx] = time
	} else {
		if correctable {
			self.timebase = pkt.Time - correcttime
			pkt.Time = correcttime
			self.time[pkt.Idx] = correcttime
		} else {
			drop = true
		}
	}

	return
}

func (self *AVSync) check(i int) (start time.Duration, end time.Duration, correctable bool, correcttime time.Duration) {
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
	end = start + self.MaxTimeDiff
	correcttime = start + time.Millisecond*40
	return
}

type Walltime struct {
	firsttime time.Time
}

func (self *Walltime) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	if pkt.Idx == 0 {
		if self.firsttime.IsZero() {
			self.firsttime = time.Now()
		}
		pkttime := self.firsttime.Add(pkt.Time)
		delta := pkttime.Sub(time.Now())
		if delta > 0 {
			time.Sleep(delta)
		}
	}
	return
}

