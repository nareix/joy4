package pktque

import (
	"github.com/nareix/joy4/av"
)

type Buf struct {
	Head, Tail    BufPos
	pkts          []av.Packet
	size, maxsize int
	count         int
}

func NewBuf() *Buf {
	return &Buf{
		pkts:    make([]av.Packet, 64),
		maxsize: 1024 * 512,
	}
}

func (self *Buf) SetMaxSize(size int) {
	self.maxsize = size
	self.shrink()
}

func (self *Buf) shrink() {
	for self.size > self.maxsize && self.count > 1 {
		i := int(self.Head) & (len(self.pkts) - 1)
		pkt := self.pkts[i]
		self.pkts[i] = av.Packet{}
		self.size -= len(pkt.Data)
		self.Head++
		self.count--
	}
}

func (self *Buf) grow() {
	newpkts := make([]av.Packet, len(self.pkts)*2)
	for i := self.Head; i.LT(self.Tail); i++ {
		newpkts[int(i)&(len(newpkts)-1)] = self.pkts[int(i)&(len(self.pkts)-1)]
	}
	self.pkts = newpkts
}

func (self *Buf) Push(pkt av.Packet) {
	if self.count == len(self.pkts) {
		self.grow()
	}
	self.pkts[int(self.Tail)&(len(self.pkts)-1)] = pkt
	self.Tail++
	self.count++
	self.size += len(pkt.Data)
	self.shrink()
}

func (self *Buf) Get(pos BufPos) av.Packet {
	return self.pkts[int(pos)&(len(self.pkts)-1)]
}

func (self *Buf) IsValidPos(pos BufPos) bool {
	return pos.GE(self.Head) && pos.LT(self.Tail)
}

type BufPos int

func (self BufPos) LT(pos BufPos) bool {
	return self-pos < 0
}

func (self BufPos) GE(pos BufPos) bool {
	return self-pos >= 0
}

func (self BufPos) GT(pos BufPos) bool {
	return self-pos > 0
}
