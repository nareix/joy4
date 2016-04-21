package ts

import (
	"bytes"
	"github.com/nareix/av"
)

type tsPacket struct {
	av.Packet
	time float64
}

type Stream struct {
	av.StreamCommon

	time float64

	pid           uint
	buf           bytes.Buffer
	payload       []byte
	peshdr        *PESHeader
	tshdr         TSHeader

	pkts []tsPacket

	demuxer   *Demuxer
	mux       *Muxer
	streamId  uint
	tsw       *TSWriter
	dataBuf   *iovec
	cacheSize int
}

func timeToPesTs(time float64) uint64 {
	return uint64(time*PTS_HZ) + PTS_HZ
}

func timeToPCR(time float64) uint64 {
	return uint64(time*PCR_HZ) + PCR_HZ
}
