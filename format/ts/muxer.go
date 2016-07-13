package ts

import (
	"bytes"
	"bufio"
	"fmt"
	"github.com/nareix/joy4/av"
	"github.com/nareix/pio"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/joy4/codec/h264parser"
	"io"
)

type Muxer struct {
	w                        *bufio.Writer
	streams                  []*Stream
	PaddingToMakeCounterCont bool

	peshdr []byte
	tshdr []byte
	adtshdr []byte
	datav [][]byte
	nalus [][]byte

	tswPAT *TSWriter
	tswPMT *TSWriter
}

var CodecTypes = []av.CodecType{av.H264, av.AAC}

func (self *Muxer) newStream(codec av.CodecData) (err error) {
	ok := false
	for _, c := range CodecTypes {
		if codec.Type() == c {
			ok = true
			break
		}
	}
	if !ok {
		err = fmt.Errorf("ts: codec type=%s is not supported", codec.Type())
		return
	}

	pid := uint(len(self.streams) + 0x100)
	stream := &Stream{
		muxer:     self,
		CodecData: codec,
		pid: pid,
		tsw: NewTSWriter(uint16(pid)),
	}
	self.streams = append(self.streams, stream)
	return
}

/*
func (self *Muxer) writePaddingTSPackets(tsw *TSWriter) (err error) {
	for tsw.ContinuityCounter&0xf != 0x0 {
		header := TSHeader{
			PID:               tsw.PID,
			ContinuityCounter: tsw.ContinuityCounter,
		}
		if _, err = WriteTSHeader(self.w, header, 0); err != nil {
			return
		}
		tsw.ContinuityCounter++
	}
	return
}
*/

func (self *Muxer) WriteTrailer() (err error) {
	if err = self.w.Flush(); err != nil {
		return
	}

	/*
	if self.PaddingToMakeCounterCont {
		for _, stream := range self.streams {
			if err = self.writePaddingTSPackets(stream.tsw); err != nil {
				return
			}
		}
	}
	*/
	return
}

func NewMuxer(w io.Writer) *Muxer {
	return &Muxer{
		w: bufio.NewWriterSize(w, pio.RecommendBufioSize),
		peshdr: make([]byte, MaxPESHeaderLength),
		tshdr: make([]byte, MaxTSHeaderLength),
		adtshdr: make([]byte, aacparser.ADTSHeaderLength),
		nalus: make([][]byte, 16),
		datav: make([][]byte, 16),
	}
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
			elemStreams = append(elemStreams, ElementaryStreamInfo{StreamType: ElementaryStreamTypeAdtsAAC, ElementaryPID: stream.pid})
		case av.H264:
			elemStreams = append(elemStreams, ElementaryStreamInfo{StreamType: ElementaryStreamTypeH264, ElementaryPID: stream.pid})
		}
	}

	pmt := PMT{
		PCRPID:                0x100,
		ElementaryStreamInfos: elemStreams,
	}
	WritePMT(bufPMT, pmt)

	self.tswPMT = NewTSWriter(0x1000)
	self.tswPAT = NewTSWriter(0)
	if err = self.tswPAT.WritePackets(self.w, [][]byte{bufPAT.Bytes()}, 0, false, true); err != nil {
		return
	}
	if err = self.tswPMT.WritePackets(self.w, [][]byte{bufPMT.Bytes()}, 0, false, true); err != nil {
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

const MaxPESHeaderLength = 19
const MaxTSHeaderLength = 12

func FillPESHeader(h []byte, streamid uint8, datalength int, pts, dts uint64) (n int) {
	h[0] = 0
	h[1] = 0
	h[2] = 1
	h[3] = streamid

	const PTS = 1 << 7
	const DTS = 1 << 6

	var pts_dts_flags uint8
	if pts != 0 {
		pts_dts_flags |= PTS
		if dts != 0 {
			pts_dts_flags |= DTS
		}
	}

	if pts_dts_flags&PTS != 0 {
		n += 5
	}
	if pts_dts_flags&DTS != 0 {
		n += 5
	}

	var packet_length uint16
	// packet_length(16) if zero then variable length
	// Specifies the number of bytes remaining in the packet after this field. Can be zero.
	// If the PES packet length is set to zero, the PES packet can be of any length.
	// A value of zero for the PES packet length can be used only when the PES packet payload is a **video** elementary stream.
	if datalength >= 0 {
		packet_length = uint16(datalength + n + 3)
	}
	pio.PutU16BE(h[4:6], packet_length)

	h[6] = 2<<6|1 // resverd(6,2)=2,original_or_copy(0,1)=1
	h[7] = pts_dts_flags
	h[8] = uint8(n)

	// pts(40)?
	// dts(40)?
	if pts_dts_flags&PTS != 0 {
		if pts_dts_flags&DTS != 0 {
			pio.PutU40BE(h[9:14], PESTsToUInt(pts)|3<<36)
			pio.PutU40BE(h[14:19], PESTsToUInt(dts)|1<<36)
		} else {
			pio.PutU40BE(h[9:14], PESTsToUInt(pts)|2<<36)
		}
	}

	n += 9
	return
}

func NewTSWriter(pid uint16) *TSWriter {
	w := &TSWriter{}
	w.tshdr = make([]byte, 188)
	w.tshdr[0] = 0x47
	pio.PutU16BE(w.tshdr[1:3], pid&0x1fff)
	for i := 6; i < 188; i++ {
		w.tshdr[i] = 0xff
	}
	return w
}

func (self *TSWriter) WritePackets(w io.Writer, datav [][]byte, pcr uint64, sync bool, paddata bool) (err error) {
	datavlen := pio.VecLen(datav)
	writev := make([][]byte, len(datav))
	writepos := 0

	for writepos < datavlen {
		self.tshdr[1] = self.tshdr[1]&0x1f
		self.tshdr[3] = byte(self.ContinuityCounter)&0xf|0x30
		self.tshdr[5] = 0 // flags
		hdrlen := 6
		self.ContinuityCounter++

		if writepos == 0 {
			self.tshdr[1] = 0x40|self.tshdr[1] // Payload Unit Start Indicator
			if pcr != 0 {
				hdrlen += 6
				self.tshdr[5] = 0x10|self.tshdr[5] // PCR flag (Discontinuity indicator 0x80)
				pio.PutU48BE(self.tshdr[6:12], PCRToUInt(pcr))
			}
			if sync {
				self.tshdr[5] = 0x40|self.tshdr[5] // Random Access indicator
			}
		}

		padtail := 0
		end := writepos + 188 - hdrlen
		if end > datavlen {
			if paddata {
				padtail = end - datavlen
			} else {
				hdrlen += end - datavlen
			}
			end = datavlen
		}
		n := pio.VecSliceNoNew(datav, writev, writepos, end)

		self.tshdr[4] = byte(hdrlen)-5 // length
		if _, err = w.Write(self.tshdr[:hdrlen]); err != nil {
			return
		}
		for i := 0; i < n; i++ {
			if _, err = w.Write(writev[i]); err != nil {
				return
			}
		}
		if padtail > 0 {
			if _, err = w.Write(self.tshdr[188-padtail:188]); err != nil {
				return
			}
		}

		writepos = end
	}

	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	stream := self.streams[pkt.Idx]

	switch stream.Type() {
	case av.AAC:
		codec := stream.CodecData.(aacparser.CodecData)

		n := FillPESHeader(self.peshdr, StreamIdAAC, len(self.adtshdr)+len(pkt.Data), timeToPesTs(pkt.Time), 0)
		self.datav[0] = self.peshdr[:n]
		aacparser.FillADTSHeader(self.adtshdr, codec.Config, 1024, len(pkt.Data))
		self.datav[1] = self.adtshdr
		self.datav[2] = pkt.Data

		if err = stream.tsw.WritePackets(self.w, self.datav[:3], timeToPCR(pkt.Time), true, false); err != nil {
			return
		}

	case av.H264:
		codec := stream.CodecData.(h264parser.CodecData)

		nalus := self.nalus[:0]
		if pkt.IsKeyFrame {
			nalus = append(nalus, codec.SPS())
			nalus = append(nalus, codec.PPS())
		}
		pktnalus, _ := h264parser.SplitNALUs(pkt.Data)
		for _, nalu := range pktnalus {
			nalus = append(nalus, nalu)
		}

		datav := self.datav[:1]
		h264parser.WalkNALUsAnnexb(nalus, func(b []byte) {
			datav = append(datav, b)
		})

		pts := timeToPesTs(pkt.Time+pkt.CompositionTime)
		dts := timeToPesTs(pkt.Time)
		n := FillPESHeader(self.peshdr, StreamIdH264, -1, pts, dts)
		datav[0] = self.peshdr[:n]

		if err = stream.tsw.WritePackets(self.w, datav, timeToPCR(pkt.Time), pkt.IsKeyFrame, false); err != nil {
			return
		}
	}

	return
}
