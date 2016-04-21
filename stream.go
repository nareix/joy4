package ts

import (
	"bytes"
	"github.com/nareix/codec/aacparser"
)

type Track struct {
	SPS []byte
	PPS []byte

	Type int

	pid       uint
	PTS       int64
	timeScale int64

	mpeg4AudioConfig aacparser.MPEG4AudioConfig
	buf              bytes.Buffer
	payload          []byte
	peshdr           *PESHeader
	tshdr            TSHeader
	spsHasWritten    bool
	payloadReady     bool

	demuxer   *Demuxer
	mux       *Muxer
	streamId  uint
	tsw       *TSWriter
	dataBuf   *iovec
	cacheSize int
}

const (
	H264 = 1
	AAC  = 2
)
