package pubsub

import (
	"github.com/nareix/joy4/av"
	"io"
	"time"
	"sync"
)

//        time
// ----------------->
// 
// V-A-V-V-A-V-V-A-V-V
// |                 |
// 0        5        10
// head             tail

type Queue struct {
	pkts []av.Packet
	head, tail int
	lock *sync.RWMutex
	cond *sync.Cond
	maxdur time.Duration
	streams []av.CodecData
	closed bool
}

func NewQueue(streams []av.CodecData) *Queue {
	q := &Queue{}
	q.streams = streams
	q.maxdur = time.Second*60
	q.lock = &sync.RWMutex{}
	q.cond = sync.NewCond(q.lock.RLocker())
	return q
}

func (self *Queue) SetMaxDuration(dur time.Duration) {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.maxdur = dur
	for self.maxdur > 0 && len(self.pkts) >= 2 && self.pkts[len(self.pkts)-1].Time - self.pkts[0].Time > self.maxdur {
		self.pkts = self.pkts[1:]
		self.head++
	}
	return
}

func (self *Queue) Duration() (dur time.Duration) {
	self.lock.RLock()
	defer self.lock.RUnlock()

	if len(self.pkts) >= 2 {
		dur = self.pkts[len(self.pkts)-1].Time - self.pkts[0].Time
	}
	return
}

func (self *Queue) Close() (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.closed = true
	self.cond.Broadcast()
	return
}

func (self *Queue) WritePacket(pkt av.Packet) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.maxdur > 0 && len(self.pkts) >= 2 && self.pkts[len(self.pkts)-1].Time - self.pkts[0].Time > self.maxdur {
		self.pkts = self.pkts[1:]
		self.head++
	}
	self.pkts = append(self.pkts, pkt)
	self.tail++
	self.cond.Broadcast()
	return
}

type QueueCursor struct {
	que *Queue
	pos int
}

func (self *Queue) Latest() *QueueCursor {
	self.lock.RLock()
	defer self.lock.RUnlock()

	return &QueueCursor{que: self, pos: self.tail}
}

func (self *Queue) Oldest() *QueueCursor {
	self.lock.RLock()
	defer self.lock.RUnlock()

	return &QueueCursor{que: self, pos: self.head}
}

func (self *Queue) Delayed(dur time.Duration) *QueueCursor {
	self.lock.RLock()
	defer self.lock.RUnlock()

	i := len(self.pkts)-1
	if i > 0 {
		end := self.pkts[i]
		for i--; i >= 0; i-- {
			if end.Time - self.pkts[i].Time > dur {
				break
			}
		}
	}

	return &QueueCursor{que: self, pos: self.head+i}
}

func (self *QueueCursor) Streams() (streams []av.CodecData, err error) {
	self.que.lock.RLock()
	defer self.que.lock.RUnlock()

	streams = self.que.streams
	return
}

func (self *QueueCursor) ReadPacket() (pkt av.Packet, err error) {
	self.que.cond.L.Lock()
	for {
		if self.pos - self.que.head < 0 {
			self.pos = self.que.head
		} else if self.pos - self.que.tail > 0 {
			self.pos = self.que.tail
		}
		if self.pos - self.que.head >= 0 && self.pos - self.que.tail < 0 {
			pkt = self.que.pkts[self.pos - self.que.head]
			self.pos++
			break
		}
		if self.que.closed {
			err = io.EOF
			break
		}
		self.que.cond.Wait()
	}
	self.que.cond.L.Unlock()
	return
}

