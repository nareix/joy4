
package ts

import (
	"bytes"
	"io"
)

type Track struct {
	timeScale int64
	writeSPS bool
	SPS []byte
	PPS []byte
	spsHasWritten bool
	tsw *TSWriter
	PTS int64
	PCR int64
	pesBuf *bytes.Buffer
}

func (self *Track) WriteH264NALU(sync bool, duration int, nalu []byte) (err error) {
	nalus := [][]byte{}

	if !self.spsHasWritten {
		nalus = append(nalus, self.SPS)
		nalus = append(nalus, self.PPS)
		self.spsHasWritten = true
	}
	nalus = append(nalus, nalu)

	pes := PESHeader{
		StreamId: StreamIdH264,
		PTS: uint64(self.PTS)*PTS_HZ/uint64(self.timeScale),
	}
	if err = WritePESHeader(self.pesBuf, pes); err != nil {
		return
	}

	data := &iovec{}
	data.Append(self.pesBuf.Bytes())
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

	self.tsw.RandomAccessIndicator = sync
	self.tsw.PCR = uint64(self.PCR)*PCR_HZ/uint64(self.timeScale)

	if err = self.tsw.WriteIovec(data); err != nil {
		return
	}

	self.PTS += int64(duration)
	self.PCR += int64(duration)
	self.pesBuf.Reset()

	return
}

func (self *Track) WriteADTSAACFrame(duration int, frame []byte) (err error) {
	pes := PESHeader{
		StreamId: StreamIdAAC,
		PTS: uint64(self.PTS)*PTS_HZ/uint64(self.timeScale),
	}
	if err = WritePESHeader(self.pesBuf, pes); err != nil {
		return
	}

	data := &iovec{}
	data.Append(self.pesBuf.Bytes())
	data.Append(frame)

	self.tsw.RandomAccessIndicator = true
	self.tsw.PCR = uint64(self.PCR)*PCR_HZ/uint64(self.timeScale)
	if err = self.tsw.WriteIovec(data); err != nil {
		return
	}

	self.PTS += int64(duration)
	self.PCR += int64(duration)
	self.pesBuf.Reset()

	return
}

func newTrack(w io.Writer, pid uint, timeScale int64) (track *Track) {
	track = &Track{
		tsw: &TSWriter{
			W: w,
			PID: pid,
			DiscontinuityIndicator: true,
		},
		timeScale: timeScale,
		pesBuf: &bytes.Buffer{},
	}
	track.tsw.EnableVecWriter()
	track.PTS = timeScale
	track.PCR = timeScale
	return
}

type Muxer struct {
	W io.Writer
	TimeScale int64
	tswPAT *TSWriter
	tswPMT *TSWriter
	elemStreams []ElementaryStreamInfo
}

func (self *Muxer) AddAACTrack() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeAdtsAAC, ElementaryPID: 0x101},
	)
	return newTrack(self.W, 0x101, self.TimeScale)
}

func (self *Muxer) AddH264Track() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeH264, ElementaryPID: 0x100},
	)
	return newTrack(self.W, 0x100, self.TimeScale)
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

