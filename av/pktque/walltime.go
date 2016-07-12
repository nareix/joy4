package pktque

import (
	"time"
	"github.com/nareix/joy4/av"
)

type WalltimeDemuxer struct {
	av.Demuxer
	firsttime time.Time
}

func (self *WalltimeDemuxer) ReadPacket() (pkt av.Packet, err error) {
	if pkt, err = self.Demuxer.ReadPacket(); err != nil {
		return
	}
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

func NewWalltimeDemuxer(demuxer av.Demuxer) *WalltimeDemuxer {
	return &WalltimeDemuxer{Demuxer: demuxer}
}

