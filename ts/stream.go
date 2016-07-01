package ts

import (
	"time"
	"bytes"
	"github.com/nareix/joy4/av"
)

type Stream struct {
	av.CodecData

	pid    uint
	buf    bytes.Buffer
	peshdr *PESHeader
	tshdr  TSHeader

	demuxer *Demuxer
	muxer   *Muxer

	streamId   uint
	streamType uint

	tsw       *TSWriter
	dataBuf   *iovec
	cacheSize int

	idx  int
}

func timeToPesTs(tm time.Duration) uint64 {
	return uint64(tm*PTS_HZ/time.Second) + PTS_HZ
}

func timeToPCR(tm time.Duration) uint64 {
	return uint64(tm*PCR_HZ/time.Second) + PCR_HZ
}
