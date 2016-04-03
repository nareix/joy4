
package ts

import (
	"bytes"
	"io"
	"github.com/nareix/mp4/isom"
)

type Muxer struct {
	W io.Writer
	tswPAT *TSWriter
	tswPMT *TSWriter
	elemStreams []ElementaryStreamInfo
	TrackH264 *Track
	Tracks []*Track
}

func (self *Muxer) newTrack(pid uint, streamId uint) (track *Track) {
	track = &Track{
		mux: self,
		tsw: &TSWriter{
			PID: pid,
			DiscontinuityIndicator: true,
		},
		streamId: streamId,
	}
	track.tsw.EnableVecWriter()
	return
}

func (self *Muxer) AddAACTrack() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeAdtsAAC, ElementaryPID: 0x101},
	)
	track = self.newTrack(0x101, StreamIdAAC)
	track.Type = AAC
	track.cacheSize = 3000
	self.Tracks = append(self.Tracks, track)
	return
}

func (self *Muxer) AddH264Track() (track *Track) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeH264, ElementaryPID: 0x100},
	)
	track = self.newTrack(0x100, StreamIdH264)
	track.Type = H264
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

func (self *Track) SetH264PPSAndSPS(pps []byte, sps []byte) {
	self.PPS, self.SPS = pps, sps
}

func (self *Track) SetTimeScale(timeScale int64) {
	self.timeScale = timeScale
}

func (self *Track) TimeScale() int64 {
	return self.timeScale
}

func (self *Track) SetMPEG4AudioConfig(config isom.MPEG4AudioConfig) {
	self.mpeg4AudioConfig = config
}

func (self *Track) tsToPesTs(ts int64) uint64 {
	return uint64(ts)*PTS_HZ/uint64(self.timeScale)
}

func (self *Track) tsToPCR(ts int64) uint64 {
	return uint64(ts)*PCR_HZ/uint64(self.timeScale)+PCR_HZ
}

func (self *Track) WriteSample(pts int64, dts int64, isKeyFrame bool, data []byte) (err error) {
	if self.Type == AAC {

		if !isom.IsADTSFrame(data) {
			data = append(data, isom.MakeADTSHeader(self.mpeg4AudioConfig, 1024, len(data))...)
		}

		buf := &bytes.Buffer{}
		pes := PESHeader{
			StreamId: self.streamId,
			PTS: self.tsToPesTs(pts),
		}
		WritePESHeader(buf, pes, len(data))
		buf.Write(data)

		self.tsw.RandomAccessIndicator = true
		self.tsw.PCR = self.tsToPCR(dts)
		if err = self.tsw.WriteTo(self.mux.W, buf.Bytes()); err != nil {
			return
		}
	} else if self.Type == H264 {

		buf := &bytes.Buffer{}
		pes := PESHeader{
			StreamId: self.streamId,
			PTS: self.tsToPesTs(pts),
		}
		if dts != pts {
			pes.DTS = self.tsToPesTs(dts)
		}
		WritePESHeader(buf, pes, 0)

		if isKeyFrame {
			buf.Write([]byte{0,0,0,1,0x9,0xf0,0,0,0,1}) // AUD
			buf.Write(self.SPS)
			buf.Write([]byte{0,0,1})
			buf.Write(self.PPS)
			buf.Write([]byte{0,0,1})
			buf.Write(data)
		} else {
			buf.Write([]byte{0,0,0,1,0x9,0xf0,0,0,0,1}) // AUD
			buf.Write(data)
		}

		self.tsw.RandomAccessIndicator = isKeyFrame
		self.tsw.PCR = self.tsToPCR(dts)
		if err = self.tsw.WriteTo(self.mux.W, buf.Bytes()); err != nil {
			return
		}
	}
	return
}

/* about to remove */

func (self *Track) setPCR() {
	self.tsw.PCR = uint64(self.PTS)*PCR_HZ/uint64(self.timeScale)
}

func (self *Track) getPesHeader(dataLength int) (data []byte){
	if self.PTS == 0 {
		self.PTS = self.timeScale
	}
	buf := &bytes.Buffer{}
	pes := PESHeader{
		StreamId: self.streamId,
		PTS: uint64(self.PTS)*PTS_HZ/uint64(self.timeScale),
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

