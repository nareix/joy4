package ts

import (
	"bytes"
	"github.com/nareix/av"
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
	pkt  av.Packet
	time float32
}

func timeToPesTs(time float32) uint64 {
	return uint64(time*PTS_HZ) + PTS_HZ
}

func timeToPCR(time float32) uint64 {
	return uint64(time*PCR_HZ) + PCR_HZ
}
