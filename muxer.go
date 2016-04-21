package ts

import (
	"bytes"
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec/h264parser"
	"io"
)

type Muxer struct {
	W       io.Writer
	streams []*Stream
}

func (self *Muxer) NewStream() av.Stream {
	stream := &Stream{
		mux: self,
		tsw: &TSWriter{
			DiscontinuityIndicator: true,
			PID: uint(len(self.streams) + 0x100),
		},
	}
	self.streams = append(self.streams, stream)
	return stream
}

func (self *Muxer) WriteTrailer() (err error) {
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

	return
}

func (self *Muxer) WritePacket(streamIndex int, pkt av.Packet) (err error) {
	stream := self.streams[streamIndex]

	if stream.Type() == av.AAC {
		data := pkt.Data
		if !aacparser.IsADTSFrame(data) {
			data = append(aacparser.MakeADTSHeader(stream.AACCodecInfo.MPEG4AudioConfig, 1024, len(data)), data...)
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
		buf := &bytes.Buffer{}
		pes := PESHeader{
			StreamId: StreamIdH264,
			PTS:      timeToPesTs(stream.time + pkt.CompositionTime),
			DTS:      timeToPesTs(stream.time),
		}
		WritePESHeader(buf, pes, 0)

		nalus, _ := h264parser.SplitNALUs(pkt.Data)
		if pkt.IsKeyFrame {
			sps := stream.H264CodecInfo.Record.SPS[0]
			pps := stream.H264CodecInfo.Record.PPS[0]
			nalus = append([][]byte{sps, pps}, nalus...)
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
