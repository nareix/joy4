package pktreorder

import (
	"github.com/nareix/av"
	"fmt"
	"io"
)

const debug = false

type stream struct {
	isVideo bool
	pkts []av.Packet
	pos float64
}

type Queue struct {
	streams []*stream
	pktnr int
	err error
}

func (self *Queue) Alloc(streams []av.CodecData) {
	self.streams = []*stream{}
	self.pktnr = 0
	for _, s := range streams {
		self.streams = append(self.streams, &stream{isVideo: s.IsVideo()})
	}
}

func (self *Queue) chooseStream() (chosen int) {
	flush := self.err != nil
	minpos := float64(-1)
	chosen = -1

	for i, stream := range self.streams {
		if (minpos < 0 || stream.pos < minpos || stream.pos == minpos && stream.isVideo) &&
			(!flush || flush && len(stream.pkts) > 0) {
			minpos = stream.pos
			chosen = i
		}
		if debug {
			fmt.Println("pktreorder: chooseStream", "flush", flush, "i", i, "pkts", len(stream.pkts))
		}
	}
	return
}

func (self *Queue) ReadPacket() (i int, pkt av.Packet, err error) {
	if self.pktnr == 0 {
		if self.err != nil {
			err = self.err
		} else {
			err = io.EOF
		}
		return
	}

	chosen := self.chooseStream()
	if chosen < 0 {
		err = io.EOF
		return
	}
	stream := self.streams[chosen]
	if len(stream.pkts) == 0 {
		err = io.EOF
		return
	}

	i = chosen
	pkt = stream.pkts[0]
	stream.pkts = stream.pkts[1:]
	stream.pos += pkt.Duration
	self.pktnr--

	return
}

func (self *Queue) WritePacket(i int, pkt av.Packet) (err error) {
	if debug {
		fmt.Println("pktreorder: WritePacket", "i", i, "Duration", fmt.Sprintf("%.2f", pkt.Duration))
	}

	stream := self.streams[i]
	stream.pkts = append(stream.pkts, pkt)
	self.pktnr++
	return
}

func (self *Queue) CanReadPacket() bool {
	chosen := self.chooseStream()
	if chosen < 0 {
		return false
	}
	if len(self.streams[chosen].pkts) == 0 {
		return false
	}
	return true
}

func (self *Queue) CanWritePacket() bool {
	return self.err == nil
}

func (self *Queue) EndWritePacket(err error) {
	if err == nil {
		err = io.EOF
	}
	self.err = err
}

func (self *Queue) Error() error {
	return self.err
}

