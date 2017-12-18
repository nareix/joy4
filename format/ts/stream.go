package ts

import (
	"time"
	"github.com/jinleileiking/joy4/av"
	"github.com/jinleileiking/joy4/format/ts/tsio"
)

type Stream struct {
	av.CodecData

	demuxer *Demuxer
	muxer   *Muxer

	pid    uint16
	streamId   uint8
	streamType uint8

	tsw       *tsio.TSWriter
	idx  int

	iskeyframe bool
	pts, dts time.Duration
	data []byte
	datalen int
}

