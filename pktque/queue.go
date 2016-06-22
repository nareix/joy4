package pktque

import (
	"github.com/nareix/av"
	"time"
)

//        time
// -------------------->
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
	size int
}

func (self *Queue) Push(pkt av.Packet) {
	if self.size == self.n {
		newsize := 0
		if self.size == 0 {
			newsize = 8
		} else {
			newsize = self.size*2
		}
		newbuf := make([]av.Packet, newsize)
		for i := 0; i < self.n; i++ {
			j := (self.head+i)%self.size
			newbuf[i] = self.buf[j]
		}
		newbuf[self.n] = pkt
		self.n++
		self.buf = newbuf
		self.size = newsize
		self.head = 0
		self.tail = self.n
	} else {
		self.buf[self.tail] = pkt
		self.tail = (self.tail+1)%self.size
		self.n++
	}
}

func (self *Queue) Pop() (pkt av.Packet) {
	if self.n == 0 {
		return
	}
	pkt = self.buf[self.head]
	self.head = (self.head+1)%self.size
	self.n--
	return
}

func (self *Queue) Head() (pkt av.Packet) {
	return self.buf[self.head]
}

func (self *Queue) Tail() (pkt av.Packet) {
	return self.buf[(self.tail-1+self.size)%self.size]
}

func (self *Queue) HeadIdx(diff int) (pkt av.Packet) {
	i := (self.head+diff)%self.size
	return self.buf[i]
}

func (self *Queue) TailIdx(diff int) (pkt av.Packet) {
	i := (self.tail-1-diff+self.size)%self.size
	return self.buf[i]
}

func (self *Queue) Count() int {
	return self.n
}

type Queues []Queue

func (self Queues) MinTimeIdx() (minidx int) {
	mintm := time.Duration(0)
	minidx = -1
	for i, que := range self {
		if que.Count() > 0 {
			headtm := que.Head().Time
			if minidx == -1 || headtm < mintm {
				minidx = i
				mintm = headtm
			}
		}
	}
	return
}

func (self Queues) RemoveBeforeTime(tm time.Duration) {
	for i := range self {
		que := &self[i]
		for que.Count() > 0 {
			if que.Head().Time < tm {
				que.Pop()
			} else {
				break
			}
		}
	}
}

