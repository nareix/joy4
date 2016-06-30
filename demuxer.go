package ts

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"
	"github.com/nareix/av"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec/h264parser"
	"io"
)

type Demuxer struct {
	R io.Reader

	gotpkt  bool
	pkt av.Packet

	pat     PAT
	pmt     *PMT
	streams []*Stream

	probed bool
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
	if err = self.poll(); err != nil {
		return
	}
	pkt = self.pkt
	return
}

func (self *Demuxer) poll() (err error) {
	for !self.gotpkt {
		if err = self.readTSPacket(); err != nil {
			return
		}
	}
	self.gotpkt = false
	return
}

func (self *Demuxer) readTSPacket() (err error) {
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
			self.streams = []*Stream{}

			for _, entry := range self.pat.Entries {
				if entry.ProgramMapPID == header.PID {
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
				if header.PID == stream.pid {
					if err = stream.handleTSPacket(header, payload); err != nil {
						return
					}
				}
			}

		}
	}

	return
}

func (self *Stream) payloadEnd() (err error) {
	payload := self.buf.Bytes()

	dts := self.peshdr.DTS
	pts := self.peshdr.PTS
	if dts == 0 {
		dts = pts
	}

	pkt := &self.demuxer.pkt
	*pkt = av.Packet{
		Idx: int8(self.idx),
		IsKeyFrame: self.tshdr.RandomAccessIndicator,
		Time: time.Duration(dts)*time.Second / time.Duration(PTS_HZ),
		Data:       payload,
	}
	if pts != dts {
		pkt.CompositionTime = time.Duration(pts-dts)*time.Second / time.Duration(PTS_HZ)
	}
	self.demuxer.gotpkt = true

	if self.CodecData == nil {
		switch self.streamType {
		case ElementaryStreamTypeAdtsAAC:
			var config aacparser.MPEG4AudioConfig
			if config, _, _, _, err = aacparser.ReadADTSFrame(payload); err != nil {
				err = fmt.Errorf("ReadADTSFrame failed: %s", err)
				return
			}
			bw := &bytes.Buffer{}
			if err = aacparser.WriteMPEG4AudioConfig(bw, config); err != nil {
				err = fmt.Errorf("WriteMPEG4AudioConfig failed: %s", err)
				return
			}
			if self.CodecData, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(bw.Bytes()); err != nil {
				return
			}

		case ElementaryStreamTypeH264:
			if false {
				fmt.Println(hex.Dump(payload))
			}
			nalus, _ := h264parser.SplitNALUs(payload)
			var sps, pps []byte
			for _, nalu := range nalus {
				if len(nalu) > 0 {
					naltype := nalu[0] & 0x1f
					if naltype == 7 {
						sps = nalu
					} else if naltype == 8 {
						pps = nalu
					}
				}
			}
			if len(sps) > 0 && len(pps) > 0 {
				if self.CodecData, err = h264parser.NewCodecDataFromSPSAndPPS(sps, pps); err != nil {
					return
				}
			}
		}
	}

	return
}

func (self *Stream) handleTSPacket(header TSHeader, tspacket []byte) (err error) {
	r := bytes.NewReader(tspacket)
	lr := &io.LimitedReader{R: r, N: int64(len(tspacket))}

	if header.PayloadUnitStart && self.peshdr != nil && self.peshdr.DataLength == 0 {
		if err = self.payloadEnd(); err != nil {
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
		if err = self.payloadEnd(); err != nil {
			return
		}
	}

	return
}
