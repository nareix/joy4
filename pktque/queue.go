package pktque

import (
	"github.com/nareix/av"
)

//        time
// 0 ----- 5 -----  10
// 
// V-A-V-V-A-V-V-A-V-V
// |                 |
// head           tail
// pop            push

type Queue struct {
	buf []av.Packet
	head int
	tail int
	n int
}

func (self *Queue) Push(pkt av.Packet) {
	if self.n == len(self.buf) {
		self.buf = append(self.buf, pkt)
		self.n++
	} else {
		self.buf[self.tail] = pkt
		self.tail = (self.tail+1)%len(self.buf)
		self.n++
	}
}

func (self *Queue) Pop() (pkt av.Packet) {
	if self.n == 0 {
		return
	}
	pkt = self.buf[self.head]
	self.head = (self.head+1)%len(self.buf)
	self.n--
	return
}

func (self *Queue) Head() (pkt av.Packet) {
	return self.buf[self.head]
}

func (self *Queue) Tail() (pkt av.Packet) {
	return self.buf[self.tail]
}

func (self *Queue) HeadIdx(diff int) (pkt av.Packet) {
	i := (self.head+diff)%len(self.buf)
	return self.buf[i]
}

func (self *Queue) TailIdx(diff int) (pkt av.Packet) {
	i := (self.tail-diff+len(self.buf))%len(self.buf)
	return self.buf[i]
}

func (self *Queue) Count() int {
	return self.n
}


