
package ts

import (
	"bytes"
	"io"
)

type Track struct {
	SPS []byte
	PPS []byte

	PTS int64
	TimeScale int64

	writeSPS bool
	spsHasWritten bool
	pcrHasWritten bool

	streamId uint
	tsw *TSWriter
	dataBuf *iovec
	cacheSize int
}

func (self *Track) setPCR() {
	if !self.pcrHasWritten {
		self.tsw.PCR = 24300000
		self.pcrHasWritten = true
	} else {
		self.tsw.PCR = 0
	}
}

func (self *Track) getPesHeader(dataLength int) (data []byte){
	if self.PTS == 0 {
		self.PTS = self.TimeScale
	}
	buf := &bytes.Buffer{}
	pes := PESHeader{
		StreamId: self.streamId,
		PTS: uint64(self.PTS)*PTS_HZ/uint64(self.TimeScale),
	}
	WritePESHeader(buf, pes, dataLength)
	return buf.Bytes()
}

func (self *Track) incPTS(delta int) {
	self.PTS += int64(delta)
}

func (self *Track) WriteH264NALU(sync bool, duration int, nalu []byte) (err error) {
	nalus := [][]byte{}

	if !self.spsHasWritten {
		nalus = append(nalus, self.SPS)
		nalus = append(nalus, self.PPS)
		self.spsHasWritten = true
	}
	nalus = append(nalus, nalu)

	data := &iovec{}
	for i, nalu := range nalus {
		var startCode []byte
		if i == 0 {
			startCode = []byte{0,0,0,1,0x9,0xf0,0,0,0,1} // AUD
		} else {
			startCode = []byte{0,0,1}
		}
		data.Append(startCode)
		data.Append(nalu)
	}

	data.Prepend(self.getPesHeader(data.Len))
	self.tsw.RandomAccessIndicator = sync
	self.setPCR()
	if err = self.tsw.WriteIovec(data); err != nil {
		return
	}

	self.incPTS(duration)
	return
}

func (self *Track) WriteADTSAACFrame(duration int, frame []byte) (err error) {
	if self.dataBuf != nil && self.dataBuf.Len > self.cacheSize {
		self.dataBuf.Prepend(self.getPesHeader(self.dataBuf.Len))
		self.tsw.RandomAccessIndicator = true
		self.setPCR()
		if err = self.tsw.WriteIovec(self.dataBuf); err != nil {
			return
		}
		self.dataBuf = nil
	}
	if self.dataBuf == nil {
		self.dataBuf = &iovec{}
	}
	self.dataBuf.Append(frame)
	self.incPTS(duration)
	return
}

func newTrack(w io.Writer, pid uint, streamId uint) (track *Track) {
	track = &Track{
		tsw: &TSWriter{
			W: w,
			PID: pid,
			DiscontinuityIndicator: true,
		},
		streamId: streamId,
	}
	track.tsw.EnableVecWriter()
	return
}

type Muxer struct {
	W io.Writer
	tswPAT *TSWriter
	tswPMT *TSWriter
	elemStreams []ElementaryStreamInfo
}

func (self *Muxer) AddAACTrack() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeAdtsAAC, ElementaryPID: 0x101},
	)
	track = newTrack(self.W, 0x101, StreamIdAAC)
	track.pcrHasWritten = true
	track.cacheSize = 3000
	return
}

func (self *Muxer) AddH264Track() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeH264, ElementaryPID: 0x100},
	)
	track = newTrack(self.W, 0x100, StreamIdH264)
	return
}

func (self *Muxer) WriteHeader() (err error) {
	bufPAT := &bytes.Buffer{}
	bufPMT := &bytes.Buffer{}

	pat := PAT{
		Entries: []PATEntry{
			{ProgramNumber: 1, ProgramMapPID: 0x1000},
		},
	}
	WritePAT(bufPAT, pat)
	pmt := PMT{
		PCRPID: 0x100,
		ElementaryStreamInfos: self.elemStreams,
	}
	WritePMT(bufPMT, pmt)

	tswPMT := &TSWriter{
		W: self.W,
		PID: 0x1000,
		DiscontinuityIndicator: true,
	}
	tswPAT := &TSWriter{
		W: self.W,
		PID: 0,
		DiscontinuityIndicator: true,
	}
	if err = tswPAT.Write(bufPAT.Bytes()); err != nil {
		return
	}
	if err = tswPMT.Write(bufPMT.Bytes()); err != nil {
		return
	}
	return
}

