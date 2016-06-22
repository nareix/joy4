package pktque

import (
	"github.com/nareix/av"
	"time"
	"fmt"
)

const debugNormalizer = false

type Normalizer struct {
	ques Queues
	streams []av.CodecData
}

func (self *Normalizer) Push(pkt av.Packet) {
}

func (self *Normalizer) Pop() (pkt av.Packet, dur time.Duration, ok bool) {
	return
}

func (self *Normalizer) Do(pkt av.Packet) (out []av.Packet) {
	const MaxDiff = time.Millisecond*800
	const MaxCacheTime = time.Second*5

	i := int(pkt.Idx)
	que := &self.ques[i]
	que.Push(pkt)

	if que.Tail().Time - que.Head().Time > MaxCacheTime {
		que.Pop()
	}

	for {
		ok := true
		diff := time.Duration(0)
		for i := range self.ques {
			if que.Count() == 0 {
				ok = false
				break
			}
			tm := self.ques[i].Tail().Time
			for j := 0; j < i; j++ {
				v := tm - self.ques[j].Tail().Time
				if v < 0 {
					v = -v
				}
				if v > diff {
					diff = v
				}
			}
		}
		if !ok {
			return
		}
		if diff > MaxDiff {
			ok = false
		}
	}

	if debugNormalizer {
		fmt.Println("normalizer: push", pkt.Idx, pkt.Time)
	}

	return
}

func NewNormalizer(streams []av.CodecData) *Normalizer {
	self := &Normalizer{}
	self.ques = make(Queues, len(streams))
	self.streams = streams
	return self
}

