package ts

import (
	"bytes"
	"fmt"
	"encoding/hex"
	"github.com/nareix/av"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec/h264parser"
	"io"
)

type Demuxer struct {
	R io.Reader

	pat     PAT
	pmt     *PMT
	streams []*Stream
	time float64

	readErr error
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
	if len(self.streams) > 0 {
		return self.streams[0].time
	}
	return 0.0
}

func (self *Demuxer) ReadHeader() (err error) {
	self.streams = []*Stream{}

	for {
		if self.pmt != nil {
			n := 0
			for _, stream := range self.streams {
				if len(stream.CodecData()) > 0 {
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
		if self.readErr != nil {
			if false {
				for _, stream := range self.streams {
					fmt.Println("read(flush): stream", stream.Type(), "pkts", len(stream.pkts))
				}
			}
			for i, stream := range self.streams {
				var ok bool
				if pkt, ok = stream.readLastPacket(); ok {
					streamIndex = i
					return
				}
			}
			err = self.readErr
			return

		} else {
			if false {
				for _, stream := range self.streams {
					fmt.Println("read(normal): stream", stream.Type(), "pkts", len(stream.pkts))
				}
			}
			for i, stream := range self.streams {
				var ok bool
				if pkt, ok = stream.readPacket(); ok {
					streamIndex = i
					return
				}
			}
		}

		if self.readErr == nil {
			self.readErr = self.readPacket()
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
					if err = stream.handleTSPacket(header, payload); err != nil {
						return
					}
				}
			}

		}
	}

	return
}

func (self *Stream) readLastPacket() (ret av.Packet, ok bool) {
	if len(self.pkts) > 1 {
		return self.readPacket()
	}
	if len(self.pkts) == 1 {
		pkt := self.pkts[0]
		self.pkts = self.pkts[1:]
		if self.peshdr.DataLength == 0 {
			pkt.Data = self.buf.Bytes()
		}
		self.time += pkt.Duration
		return pkt.Packet, true
	}
	return
}

func (self *Stream) readPacket() (ret av.Packet, ok bool) {
	if len(self.pkts) > 1 {
		pkt := self.pkts[0]
		self.pkts = self.pkts[1:]
		self.time += pkt.Duration
		return pkt.Packet, true
	}
	return
}

func (self *Stream) payloadStart() {
	dts := self.peshdr.DTS
	pts := self.peshdr.PTS
	if dts == 0 {
		dts = pts
	}

	pkt := tsPacket{
		Packet: av.Packet{
			IsKeyFrame: self.tshdr.RandomAccessIndicator,
		},
		time: float64(dts)/float64(PTS_HZ),
	}
	if pts != dts {
		pkt.CompositionTime = float64(pts-dts)/float64(PTS_HZ)
	}

	if len(self.pkts) > 0 {
		lastpkt := &self.pkts[len(self.pkts)-1]
		lastpkt.Duration = pkt.time - lastpkt.time
		self.lastDuration = lastpkt.Duration
	} else {
		pkt.Duration = self.lastDuration
	}

	self.pkts = append(self.pkts, pkt)
}

func (self *Stream) payloadEnd() (err error) {
	payload := self.buf.Bytes()

	curpkt := &self.pkts[len(self.pkts)-1]
	curpkt.Data = payload

	if len(self.CodecData()) == 0 {
		if self.Type() == av.AAC {
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
			if err = self.SetCodecData(bw.Bytes()); err != nil {
				err = fmt.Errorf("SetCodecData failed: %s", err)
				return
			}
		} else if self.Type() == av.H264 {
			if false {
				fmt.Println(hex.Dump(payload))
			}
			nalus, _ := h264parser.SplitNALUs(payload)
			var sps, pps []byte
			for _, nalu := range nalus {
				if len(nalu) > 0 {
					naltype := nalu[0]&0x1f
					if naltype == 7 {
						sps = nalu
					} else if naltype == 8 {
						pps = nalu
					}
				}
			}
			if len(sps) > 0 && len(pps) > 0 {
				codecData, _ := h264parser.CreateCodecDataBySPSAndPPS(sps, pps)
				if err = self.SetCodecData(codecData); err != nil {
					err = fmt.Errorf("SetCodecData failed: %s", err)
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
		self.payloadStart()
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
