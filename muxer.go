package ts

import (
	"bytes"
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec/h264parser"
	"io"
)

func Create(W io.Writer, streams []av.CodecData) (muxer *Muxer, err error) {
	_muxer := &Muxer{W: W}
	if err = _muxer.WriteHeader(streams); err != nil {
		return
	}
	muxer = _muxer
	return
}

type Muxer struct {
	W                        io.Writer
	streams                  []*Stream
	PaddingToMakeCounterCont bool

	tswPAT *TSWriter
	tswPMT *TSWriter
}

func (self *Muxer) isCodecSupported(codec av.CodecData) bool {
	switch codec.Type() {
	case av.H264, av.AAC:
		return true
	default:
		return false
	}
}

func (self *Muxer) newStream(codec av.CodecData) (err error) {
	if !self.isCodecSupported(codec) {
		err = fmt.Errorf("codec type=%x is not supported", codec.Type())
		return
	}

	stream := &Stream{
		muxer:     self,
		CodecData: codec,
		tsw: &TSWriter{
			DiscontinuityIndicator: true,
			PID: uint(len(self.streams) + 0x100),
		},
	}
	self.streams = append(self.streams, stream)
	return
}

func (self *Muxer) writePaddingTSPackets(tsw *TSWriter) (err error) {
	for tsw.ContinuityCounter&0xf != 0x0 {
		header := TSHeader{
			PID:               tsw.PID,
			ContinuityCounter: tsw.ContinuityCounter,
		}
		if _, err = WriteTSHeader(self.W, header, 0); err != nil {
			return
		}
		tsw.ContinuityCounter++
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {
	if self.PaddingToMakeCounterCont {
		for _, stream := range self.streams {
			if err = self.writePaddingTSPackets(stream.tsw); err != nil {
				return
			}
		}
	}
	return
}

func (self *Muxer) WritePATPMT() (err error) {
	bufPAT := &bytes.Buffer{}
	bufPMT := &bytes.Buffer{}

	pat := PAT{
		Entries: []PATEntry{
			{ProgramNumber: 1, ProgramMapPID: 0x1000},
		},
	}
	WritePAT(bufPAT, pat)

	var elemStreams []ElementaryStreamInfo
	for _, stream := range self.streams {
		switch stream.Type() {
		case av.AAC:
			elemStreams = append(elemStreams, ElementaryStreamInfo{StreamType: ElementaryStreamTypeAdtsAAC, ElementaryPID: stream.tsw.PID})
		case av.H264:
			elemStreams = append(elemStreams, ElementaryStreamInfo{StreamType: ElementaryStreamTypeH264, ElementaryPID: stream.tsw.PID})
		}
	}

	pmt := PMT{
		PCRPID:                0x100,
		ElementaryStreamInfos: elemStreams,
	}
	WritePMT(bufPMT, pmt)

	self.tswPMT = &TSWriter{
		PID: 0x1000,
		DiscontinuityIndicator: true,
	}
	self.tswPAT = &TSWriter{
		PID: 0,
		DiscontinuityIndicator: true,
	}
	if err = self.tswPAT.WriteTo(self.W, bufPAT.Bytes()); err != nil {
		return
	}
	if err = self.tswPMT.WriteTo(self.W, bufPMT.Bytes()); err != nil {
		return
	}

	return
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	self.streams = []*Stream{}
	for _, stream := range streams {
		if err = self.newStream(stream); err != nil {
			return
		}
	}

	if err = self.WritePATPMT(); err != nil {
		return
	}

	return
}

func (self *Muxer) WritePacket(streamIndex int, pkt av.Packet) (err error) {
	stream := self.streams[streamIndex]

	if stream.Type() == av.AAC {
		codec := stream.CodecData.(aacparser.CodecData)
		data := pkt.Data
		if !aacparser.IsADTSFrame(data) {
			data = append(codec.MakeADTSHeader(1024, len(data)), data...)
		}

		buf := &bytes.Buffer{}
		pes := PESHeader{
			StreamId: StreamIdAAC,
			PTS:      timeToPesTs(stream.time),
		}
		WritePESHeader(buf, pes, len(data))
		buf.Write(data)

		stream.tsw.RandomAccessIndicator = true
		stream.tsw.PCR = timeToPCR(stream.time)
		if err = stream.tsw.WriteTo(self.W, buf.Bytes()); err != nil {
			return
		}

		stream.time += pkt.Duration

	} else if stream.Type() == av.H264 {
		codec := stream.CodecData.(h264parser.CodecData)
		buf := &bytes.Buffer{}
		pes := PESHeader{
			StreamId: StreamIdH264,
			PTS:      timeToPesTs(stream.time + pkt.CompositionTime),
			DTS:      timeToPesTs(stream.time),
		}
		WritePESHeader(buf, pes, 0)

		nalus, _ := h264parser.SplitNALUs(pkt.Data)
		if pkt.IsKeyFrame {
			nalus = append([][]byte{codec.SPS(), codec.PPS()}, nalus...)
		}
		h264parser.WalkNALUsAnnexb(nalus, func(b []byte) {
			buf.Write(b)
		})

		stream.tsw.RandomAccessIndicator = pkt.IsKeyFrame
		stream.tsw.PCR = timeToPCR(stream.time)
		if err = stream.tsw.WriteTo(self.W, buf.Bytes()); err != nil {
			return
		}

		stream.time += pkt.Duration

	} else {
		err = fmt.Errorf("unknown stream type=%d", stream.Type())
		return
	}

	return
}
