package ts

import (
	"time"
	"bytes"
	"github.com/nareix/joy4/av"
)

type Stream struct {
	av.CodecData

	buf bytes.Buffer
	peshdr *PESHeader

	demuxer *Demuxer
	muxer   *Muxer
	iskeyframe bool

	pid    uint
	streamId   uint
	streamType uint

	tsw       *TSWriter
	idx  int
}

func timeToPesTs(tm time.Duration) uint64 {
	return uint64(tm*PTS_HZ/time.Second) + PTS_HZ
}

func timeToPCR(tm time.Duration) uint64 {
	return uint64(tm*PCR_HZ/time.Second) + PCR_HZ
}
