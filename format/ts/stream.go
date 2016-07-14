package ts

import (
	"time"
	"bytes"
	"github.com/nareix/joy4/av"
)

type Stream struct {
	av.CodecData

	buf bytes.Buffer

	demuxer *Demuxer
	muxer   *Muxer

	pid    uint16
	streamId   uint8
	streamType uint8

	tsw       *TSWriter
	idx  int

	iskeyframe bool
	pts, dts time.Duration
	data []byte
}

func timeToPesTs(tm time.Duration) uint64 {
	return uint64(tm*PTS_HZ/time.Second) + PTS_HZ
}

func timeToPCR(tm time.Duration) uint64 {
	return uint64(tm*PCR_HZ/time.Second) + PCR_HZ
}
