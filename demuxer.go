package ts

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"
	"github.com/nareix/av"
	"github.com/nareix/av/pktqueue"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec/h264parser"
	"io"
)

func Open(R io.Reader) (demuxer *Demuxer, err error) {
	_demuxer := &Demuxer{R: R}
	if err = _demuxer.ReadHeader(); err != nil {
		return
	}
	demuxer = _demuxer
	return
}

type Demuxer struct {
	R io.Reader

	gotpkt  bool
	pktque  *pktqueue.Queue
	pat     PAT
	pmt     *PMT
	streams []*Stream
}

func (self *Demuxer) Streams() (streams []av.CodecData, err error) {
	if len(self.streams) == 0 {
		err = fmt.Errorf("ts: no streams")
		return
	}
	for _, stream := range self.streams {
		streams = append(streams, stream)
	}
	return
}

func (self *Demuxer) CurrentTime() (tm time.Duration) {
	if self.pktque != nil {
		tm = self.pktque.CurrentTime()
	}
	return
}

func (self *Demuxer) ReadHeader() (err error) {
	self.streams = []*Stream{}

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

	return
}

func (self *Demuxer) ReadPacket() (i int, pkt av.Packet, err error) {
	return self.pktque.ReadPacket()
}

func (self *Demuxer) poll() (err error) {
	for self.gotpkt {
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
						switch info.StreamType {
						case ElementaryStreamTypeH264:
							self.streams = append(self.streams, stream)
						case ElementaryStreamTypeAdtsAAC:
							self.streams = append(self.streams, stream)
						}
					}
					self.pktque = &pktqueue.Queue{}

					var streams []av.CodecData
					if streams, err = self.Streams(); err != nil {
						return
					}
					self.pktque.Alloc(streams)
					self.pktque.Poll = self.poll
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

	pkt := av.Packet{
		IsKeyFrame: self.tshdr.RandomAccessIndicator,
		Data:       payload,
	}
	tm := time.Duration(dts)*time.Second / time.Duration(PTS_HZ)
	if pts != dts {
		pkt.CompositionTime = time.Duration(pts-dts)*time.Second / time.Duration(PTS_HZ)
	}
	self.demuxer.pktque.WriteTimePacket(self.idx, tm, pkt)
	self.demuxer.gotpkt = true

	if self.CodecData == nil {
		if self.streamType == ElementaryStreamTypeAdtsAAC {
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
		} else if self.streamType == ElementaryStreamTypeH264 {
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
