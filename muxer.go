package ts

import (
	"bytes"
	"fmt"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec/h264parser"
	"io"
)

type Muxer struct {
	W           io.Writer
	tswPAT      *TSWriter
	tswPMT      *TSWriter
	elemStreams []ElementaryStreamInfo
	TrackH264   *Stream
	Tracks      []*Stream
}

func (self *Muxer) newTrack(pid uint, streamId uint) (stream *Stream) {
	stream = &Stream{
		mux: self,
		tsw: &TSWriter{
			PID: pid,
			DiscontinuityIndicator: true,
		},
		streamId: streamId,
	}
	stream.tsw.EnableVecWriter()
	return
}

func (self *Muxer) AddAACTrack() (stream *Stream) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeAdtsAAC, ElementaryPID: 0x101},
	)
	stream = self.newTrack(0x101, StreamIdAAC)
	stream.Type = AAC
	stream.cacheSize = 3000
	self.Tracks = append(self.Tracks, stream)
	return
}

func (self *Muxer) AddH264Track() (stream *Stream) {
	self.elemStreams = append(
		self.elemStreams,
		ElementaryStreamInfo{StreamType: ElementaryStreamTypeH264, ElementaryPID: 0x100},
	)
	stream = self.newTrack(0x100, StreamIdH264)
	stream.Type = H264
	self.TrackH264 = stream
	self.Tracks = append(self.Tracks, stream)
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
		PCRPID:                0x100,
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

	// about to remove
	for _, stream := range self.Tracks {
		stream.spsHasWritten = false
	}

	return
}

func (self *Stream) SetH264PPSAndSPS(pps []byte, sps []byte) {
	self.PPS, self.SPS = pps, sps
}

func (self *Stream) SetTimeScale(timeScale int64) {
	self.timeScale = timeScale
}

func (self *Stream) TimeScale() int64 {
	return self.timeScale
}

func (self *Stream) SetMPEG4AudioConfig(config aacparser.MPEG4AudioConfig) {
	self.mpeg4AudioConfig = config
}

func (self *Stream) tsToPesTs(ts int64) uint64 {
	return uint64(ts)*PTS_HZ/uint64(self.timeScale) + PTS_HZ
}

func (self *Stream) tsToPCR(ts int64) uint64 {
	return uint64(ts)*PCR_HZ/uint64(self.timeScale) + PCR_HZ
}

func (self *Stream) tsToTime(ts int64) float64 {
	return float64(ts) / float64(self.timeScale)
}

func (self *Stream) WriteSample(pts int64, dts int64, isKeyFrame bool, data []byte) (err error) {
	if false {
		fmt.Println("WriteSample", self.Type, self.tsToTime(dts))
	}

	if self.Type == AAC {

		if !aacparser.IsADTSFrame(data) {
			data = append(aacparser.MakeADTSHeader(self.mpeg4AudioConfig, 1024, len(data)), data...)
		}
		if false {
			fmt.Printf("WriteSample=%x\n", data[:5])
		}

		buf := &bytes.Buffer{}
		pes := PESHeader{
			StreamId: self.streamId,
			PTS:      self.tsToPesTs(pts),
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
			PTS:      self.tsToPesTs(pts),
		}
		if dts != pts {
			pes.DTS = self.tsToPesTs(dts)
		}
		WritePESHeader(buf, pes, 0)

		nalus, _ := h264parser.SplitNALUs(data)
		if isKeyFrame {
			nalus = append([][]byte{self.SPS, self.PPS}, nalus...)
		}
		h264parser.WalkNALUsAnnexb(nalus, func(b []byte) {
			buf.Write(b)
		})

		self.tsw.RandomAccessIndicator = isKeyFrame
		self.tsw.PCR = self.tsToPCR(dts)
		if err = self.tsw.WriteTo(self.mux.W, buf.Bytes()); err != nil {
			return
		}
	}

	return
}

/* about to remove */

func (self *Stream) setPCR() {
	self.tsw.PCR = uint64(self.PTS) * PCR_HZ / uint64(self.timeScale)
}

func (self *Stream) getPesHeader(dataLength int) (data []byte) {
	if self.PTS == 0 {
		self.PTS = self.timeScale
	}
	buf := &bytes.Buffer{}
	pes := PESHeader{
		StreamId: self.streamId,
		PTS:      uint64(self.PTS) * PTS_HZ / uint64(self.timeScale),
	}
	WritePESHeader(buf, pes, dataLength)
	return buf.Bytes()
}

func (self *Stream) incPTS(delta int) {
	self.PTS += int64(delta)
}

func (self *Stream) WriteH264NALU(sync bool, duration int, nalu []byte) (err error) {
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
			startCode = []byte{0, 0, 0, 1, 0x9, 0xf0, 0, 0, 0, 1} // AUD
		} else {
			startCode = []byte{0, 0, 1}
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

func (self *Stream) WriteADTSAACFrame(duration int, frame []byte) (err error) {
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
