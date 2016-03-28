
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

	mux *Muxer
	streamId uint
	tsw *TSWriter
	dataBuf *iovec
	cacheSize int
}

func (self *Track) setPCR() {
	self.tsw.PCR = uint64(self.PTS)*PCR_HZ/uint64(self.TimeScale)
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

	data.Prepend(self.getPesHeader(0))
	self.tsw.RandomAccessIndicator = sync
	self.setPCR()
	if err = self.tsw.WriteIovecTo(self.mux.W, data); err != nil {
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
		if err = self.tsw.WriteIovecTo(self.mux.W, self.dataBuf); err != nil {
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

func newTrack(mux *Muxer, pid uint, streamId uint) (track *Track) {
	track = &Track{
		mux: mux,
		tsw: &TSWriter{
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
	TrackH264 *Track
	Tracks []*Track
}

func (self *Muxer) AddAACTrack() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeAdtsAAC, ElementaryPID: 0x101},
	)
	track = newTrack(self, 0x101, StreamIdAAC)
	track.cacheSize = 3000
	self.Tracks = append(self.Tracks, track)
	return
}

func (self *Muxer) AddH264Track() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeH264, ElementaryPID: 0x100},
	)
	track = newTrack(self, 0x100, StreamIdH264)
	self.TrackH264 = track
	self.Tracks = append(self.Tracks, track)
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
		PID: 0x1000,
		DiscontinuityIndicator: true,
	}
	tswPAT := &TSWriter{
		PID: 0,
		DiscontinuityIndicator: true,
	}
	if err = tswPAT.WriteTo(self.W, bufPAT.Bytes()); err != nil {
		return
	}
	if err = tswPMT.WriteTo(self.W, bufPMT.Bytes()); err != nil {
		return
	}

	for _, track := range(self.Tracks) {
		track.spsHasWritten = false
	}

	return
}

