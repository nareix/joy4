
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

type FixTime struct {
	zerobase time.Duration
	incrbase time.Duration
	lasttime time.Duration
	StartFromZero bool
	MakeIncrement bool
}

func (self *FixTime) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	if self.StartFromZero {
		if self.zerobase == 0 {
			self.zerobase = pkt.Time
		}
		pkt.Time -= self.zerobase
	}

	if self.MakeIncrement {
		pkt.Time -= self.incrbase
		if self.lasttime == 0 {
			self.lasttime = pkt.Time
		}
		if pkt.Time < self.lasttime || pkt.Time > self.lasttime+time.Millisecond*500 {
			self.incrbase += pkt.Time - self.lasttime
			pkt.Time = self.lasttime
		}
		self.lasttime = pkt.Time
	}

	return
}

type AVSync struct {
	MaxTimeDiff time.Duration
	time []time.Duration
}

func (self *AVSync) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	if self.time == nil {
		self.time = make([]time.Duration, len(streams))
		if self.MaxTimeDiff == 0 {
			self.MaxTimeDiff = time.Millisecond*500
		}
	}

	start, end, correctable, correcttime := self.check(int(pkt.Idx))
	if pkt.Time >= start && pkt.Time < end {
		self.time[pkt.Idx] = pkt.Time
	} else {
		if correctable {
			pkt.Time = correcttime
			for i := range self.time {
				self.time[i] = correcttime
			}
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

	if i == maxidx {
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

