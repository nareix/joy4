package flv

import (
	_ "fmt"
	"github.com/nareix/av"
	"github.com/nareix/pio"
	"github.com/nareix/flv/flvio"
	"io"
)

type Muxer struct {
	pw *pio.Writer
}

func NewMuxer(w io.Writer) *Muxer {
	self := &Muxer{}
	self.pw = pio.NewWriter(w)
	return self
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	hasVideo := false
	hasAudio := false
	for _, stream := range streams {
		if stream.IsVideo() {
			hasVideo = true
		} else if stream.IsAudio() {
			hasAudio = true
		}
	}

	if err = flvio.WriteFileHeader(self.pw, hasVideo, hasAudio); err != nil {
		return
	}

	return
}

func (self *Muxer) WritePacket(i int, pkt av.Packet) (err error) {
	return
}

