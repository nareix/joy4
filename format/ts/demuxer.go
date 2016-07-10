package ts

import (
	"bufio"
	"bytes"
	"fmt"
	"time"
	"github.com/nareix/pio"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/joy4/codec/h264parser"
	"io"
)

type Demuxer struct {
	r *bufio.Reader

	pktidx int
	pkts []av.Packet

	pat     PAT
	pmt     *PMT
	streams []*Stream
	tshdr []byte

	probed bool
}

func NewDemuxer(r io.Reader) *Demuxer {
	return &Demuxer{
		tshdr: make([]byte, 188),
		r: bufio.NewReaderSize(r, pio.RecommendBufioSize),
	}
}

func (self *Demuxer) Streams() (streams []av.CodecData, err error) {
	if err = self.probe(); err != nil {
		return
	}
	for _, stream := range self.streams {
		streams = append(streams, stream.CodecData)
	}
	return
}

func (self *Demuxer) probe() (err error) {
	if self.probed {
		return
	}
	for {
		if self.pmt != nil {
			n := 0
			for _, stream := range self.streams {
				if stream.CodecData != nil {
					n++
				}
			}
			if n == len(self.streams) {
				break
			}
		}
		if err = self.poll(); err != nil {
			return
		}
	}
	self.probed = true
	return
}

func (self *Demuxer) ReadPacket() (pkt av.Packet, err error) {
	if err = self.probe(); err != nil {
		return
	}

	for self.pktidx == len(self.pkts) {
		if err = self.poll(); err != nil {
			return
		}
	}
	if self.pktidx < len(self.pkts) {
		pkt = self.pkts[self.pktidx]
		self.pktidx++
	}

	return
}

func (self *Demuxer) poll() (err error) {
	self.pktidx = 0
	self.pkts = self.pkts[:0]
	for {
		if err = self.readTSPacket(); err != nil {
			return
		}
		if len(self.pkts) > 0 {
			break
		}
	}
	return
}

func ParseTSHeader(tshdr []byte) (pid uint, start bool, iskeyframe bool, hdrlen int, err error) {
	// https://en.wikipedia.org/wiki/MPEG_transport_stream
	if tshdr[0] != 0x47 {
		err = fmt.Errorf("tshdr sync invalid")
		return
	}
	pid = uint((tshdr[1]&0x1f))<<8|uint(tshdr[2])
	start = tshdr[1]&0x40 != 0
	hdrlen += 4
	if tshdr[3]&0x20 != 0 {
		hdrlen += int(tshdr[4])+1
		iskeyframe = tshdr[5]&0x40 != 0
	}
	return
}

func (self *Demuxer) readTSPacket() (err error) {
	var hdrlen int
	var pid uint
	var start bool
	var iskeyframe bool

	if _, err = io.ReadFull(self.r, self.tshdr); err != nil {
		return
	}
	if pid, start, iskeyframe, hdrlen, err = ParseTSHeader(self.tshdr); err != nil {
		return
	}
	payload := self.tshdr[hdrlen:]

	if pid == 0 {
		if self.pat, err = ReadPAT(bytes.NewReader(payload)); err != nil {
			err = fmt.Errorf("ts: invalid pat")
			return
		}
	} else {
		if self.pmt == nil {
			self.streams = []*Stream{}

			for _, entry := range self.pat.Entries {
				if entry.ProgramMapPID == pid {
					self.pmt = new(PMT)
					if *self.pmt, err = ReadPMT(bytes.NewReader(payload)); err != nil {
						return
					}
					for i, info := range self.pmt.ElementaryStreamInfos {
						stream := &Stream{}
						stream.idx = i
						stream.demuxer = self
						stream.pid = info.ElementaryPID
						stream.streamType = info.StreamType
						switch info.StreamType {
						case ElementaryStreamTypeH264:
							self.streams = append(self.streams, stream)
						case ElementaryStreamTypeAdtsAAC:
							self.streams = append(self.streams, stream)
						}
					}
				}
			}

		} else {
			for _, stream := range self.streams {
				if pid == stream.pid {
					if err = stream.handleTSPacket(start, iskeyframe, payload); err != nil {
						return
					}
				}
			}

		}
	}

	return
}

func (self *Stream) addPacket(payload []byte, timedelta time.Duration) {
	dts := self.peshdr.DTS
	pts := self.peshdr.PTS
	if dts == 0 {
		dts = pts
	}

	demuxer := self.demuxer
	pkt := av.Packet{
		Idx: int8(self.idx),
		IsKeyFrame: self.iskeyframe,
		Time: time.Duration(dts)*time.Second / time.Duration(PTS_HZ) + timedelta,
		Data:       payload,
	}
	if pts != dts {
		pkt.CompositionTime = time.Duration(pts-dts)*time.Second / time.Duration(PTS_HZ)
	}
	demuxer.pkts = append(demuxer.pkts, pkt)
}

func (self *Stream) payloadEnd() (err error) {
	payload := self.buf.Bytes()

	switch self.streamType {
	case ElementaryStreamTypeAdtsAAC:
		var config aacparser.MPEG4AudioConfig

		delta := time.Duration(0)
		for len(payload) > 0 {
			var hdrlen, framelen, samples int
			if config, hdrlen, framelen, samples, err = aacparser.ParseADTSHeader(payload); err != nil {
				return
			}
			if self.CodecData == nil {
				if self.CodecData, err = aacparser.NewCodecDataFromMPEG4AudioConfig(config); err != nil {
					return
				}
			}
			self.addPacket(payload[hdrlen:framelen], delta)
			delta += time.Duration(samples) * time.Second / time.Duration(config.SampleRate)
			payload = payload[framelen:]
		}

	case ElementaryStreamTypeH264:
		nalus, _ := h264parser.SplitNALUs(payload)
		var sps, pps []byte
		for _, nalu := range nalus {
			if len(nalu) > 0 {
				naltype := nalu[0] & 0x1f
				switch {
				case naltype == 7:
					sps = nalu
				case naltype == 8:
					pps = nalu
				case h264parser.IsDataNALU(nalu):
					// raw nalu to avcc
					b := make([]byte, 4+len(nalu))
					pio.PutU32BE(b[0:4], uint32(len(nalu)))
					copy(b[4:], nalu)
					self.addPacket(b, time.Duration(0))
				}
			}
		}

		if self.CodecData == nil && len(sps) > 0 && len(pps) > 0 {
			if self.CodecData, err = h264parser.NewCodecDataFromSPSAndPPS(sps, pps); err != nil {
				return
			}
		}
	}

	return
}

func (self *Stream) handleTSPacket(start bool, iskeyframe bool, payload []byte) (err error) {
	r := bytes.NewReader(payload)
	lr := &io.LimitedReader{R: r, N: int64(len(payload))}

	if start && self.peshdr != nil && self.peshdr.DataLength == 0 {
		if err = self.payloadEnd(); err != nil {
			return
		}
	}

	if start {
		self.buf = bytes.Buffer{}
		if self.peshdr, err = ReadPESHeader(lr); err != nil {
			return
		}
		self.iskeyframe = iskeyframe
	}

	if _, err = io.CopyN(&self.buf, lr, lr.N); err != nil {
		return
	}

	if self.buf.Len() == int(self.peshdr.DataLength) {
		if err = self.payloadEnd(); err != nil {
			return
		}
	}

	return
}
