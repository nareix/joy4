package ts

import (
	"bytes"
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/codec/aacparser"
	"io"
)

type Demuxer struct {
	R io.Reader

	pat     PAT
	pmt     *PMT
	streams []*Stream
}

// ParsePacket() (pid uint, counter int, isStart bool, pts, dst int64, isKeyFrame bool)
// WritePayload(pid, pts, dts, isKeyFrame, payloads, isVideoFrame)

func (self *Demuxer) Streams() (streams []av.Stream) {
	for _, stream := range self.streams {
		streams = append(streams, stream)
	}
	return
}

func (self *Demuxer) Time() float64 {
	for _, stream := range self.streams {
		if len(stream.pkts) > 0 {
			return stream.pkts[len(stream.pkts)-1].time
		}
	}
	return 0.0
}

func (self *Demuxer) ReadHeader() (err error) {
	self.streams = []*Stream{}

	for {
		if self.pmt != nil {
			n := 0
			for _, stream := range self.streams {
				if len(stream.pkts) > 0 {
					n++
				}
			}
			if n == len(self.streams) {
				break
			}
		}

		if err = self.readPacket(); err != nil {
			return
		}
	}

	return
}

func (self *Demuxer) ReadPacket() (streamIndex int, pkt av.Packet, err error) {
	if len(self.streams) == 0 {
		err = fmt.Errorf("no stream")
		return
	}

	for {
		for i, stream := range self.streams {
			if len(stream.pkts) > 1 {
				streamIndex = i
				pkt = stream.pkts[0].Packet
				stream.pkts = stream.pkts[1:]
				return
			}
		}

		if err = self.readPacket(); err != nil {
			return
		}
	}
}

func (self *Demuxer) readPacket() (err error) {
	var header TSHeader
	var n int
	var data [188]byte

	if header, n, err = ReadTSPacket(self.R, data[:]); err != nil {
		return
	}
	payload := data[:n]

	if header.PID == 0 {
		if self.pat, err = ReadPAT(bytes.NewReader(payload)); err != nil {
			return
		}
	} else {
		if self.pmt == nil {
			for _, entry := range self.pat.Entries {
				if entry.ProgramMapPID == header.PID {
					self.pmt = new(PMT)
					if *self.pmt, err = ReadPMT(bytes.NewReader(payload)); err != nil {
						return
					}
					for _, info := range self.pmt.ElementaryStreamInfos {
						stream := &Stream{}

						stream.demuxer = self
						stream.pid = info.ElementaryPID
						switch info.StreamType {
						case ElementaryStreamTypeH264:
							stream.SetType(av.H264)
							self.streams = append(self.streams, stream)
						case ElementaryStreamTypeAdtsAAC:
							stream.SetType(av.AAC)
							self.streams = append(self.streams, stream)
						}
					}
				}
			}

		} else {
			for _, stream := range self.streams {
				if header.PID == stream.pid {
					if err = stream.appendPacket(header, payload); err != nil {
						return
					}
				}
			}

		}
	}

	return
}

func (self *Stream) appendPayload() (err error) {
	self.payload = self.buf.Bytes()

	if self.Type() == av.AAC {
		if len(self.CodecData()) == 0 {
			var config aacparser.MPEG4AudioConfig
			if config, _, _, _, err = aacparser.ReadADTSFrame(self.payload); err != nil {
				err = fmt.Errorf("ReadADTSFrame failed: %s", err)
				return
			}
			bw := &bytes.Buffer{}
			if err = aacparser.WriteMPEG4AudioConfig(bw, config); err != nil {
				err = fmt.Errorf("WriteMPEG4AudioConfig failed: %s", err)
				return
			}
			if err = self.SetCodecData(bw.Bytes()); err != nil {
				err = fmt.Errorf("SetCodecData failed: %s", err)
				return
			}
		}
	}

	dts := self.peshdr.DTS
	pts := self.peshdr.PTS
	if dts == 0 {
		dts = pts
	}

	pkt := tsPacket{
		Packet: av.Packet{
			IsKeyFrame: self.tshdr.RandomAccessIndicator,
			Data: self.payload,
		},
		time: float64(dts)/float64(PTS_HZ),
	}

	if pts != dts {
		pkt.CompositionTime = float64(pts-dts)/float64(PTS_HZ)
	}

	if len(self.pkts) > 0 {
		lastPkt := &self.pkts[len(self.pkts)-1]
		lastPkt.Duration = pkt.time - lastPkt.time
	}
	self.pkts = append(self.pkts, pkt)

	return
}

func (self *Stream) appendPacket(header TSHeader, payload []byte) (err error) {
	r := bytes.NewReader(payload)
	lr := &io.LimitedReader{R: r, N: int64(len(payload))}

	if header.PayloadUnitStart && self.peshdr != nil && self.peshdr.DataLength == 0 {
		if err = self.appendPayload(); err != nil {
			return
		}
	}

	if header.PayloadUnitStart {
		self.buf = bytes.Buffer{}
		if self.peshdr, err = ReadPESHeader(lr); err != nil {
			return
		}
		self.tshdr = header
	}

	if _, err = io.CopyN(&self.buf, lr, lr.N); err != nil {
		return
	}

	if self.buf.Len() == int(self.peshdr.DataLength) {
		if err = self.appendPayload(); err != nil {
			return
		}
	}

	return
}
