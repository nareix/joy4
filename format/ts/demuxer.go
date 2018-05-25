package ts

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/jinleileiking/joy4/av"
	"github.com/jinleileiking/joy4/codec/aacparser"
	"github.com/jinleileiking/joy4/codec/h264parser"
	"github.com/jinleileiking/joy4/format/ts/tsio"
	"github.com/jinleileiking/joy4/utils/bits/pio"
)

type TPayloadInfos struct {
	Pts         int
	Dts         int
	PayloadInfo []byte
}

type Demuxer struct {
	r *bufio.Reader

	pkts []av.Packet

	Pat          *tsio.PAT
	Pmt          *tsio.PMT
	Payloads     [][]byte
	PayloadInfos []TPayloadInfos

	pat     *tsio.PAT
	pmt     *tsio.PMT
	streams []*Stream
	tshdr   []byte

	stage int
}

func NewDemuxer(r io.Reader) *Demuxer {
	return &Demuxer{
		tshdr: make([]byte, 188),
		r:     bufio.NewReaderSize(r, pio.RecommendBufioSize),
	}
}

func (self *Demuxer) Streams() (streams []av.CodecData, err error) {
	if err = self.probe(); err != nil {
		// fmt.Println("probe error")
		// spew.Dump("stream probe error", err)
		// return
	}

	// dump streams to return
	for _, stream := range self.streams {
		streams = append(streams, stream.CodecData)
	}
	return
}

// parse all data
func (self *Demuxer) probe() (err error) {
	if self.stage == 0 {
		for {
			//found pmt
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
		self.stage++
	}
	return
}

func (self *Demuxer) ReadPacket() (pkt av.Packet, err error) {
	if err = self.probe(); err != nil {
		return
	}

	for len(self.pkts) == 0 {
		if err = self.poll(); err != nil {
			return
		}
	}

	pkt = self.pkts[0]
	self.pkts = self.pkts[1:]
	return
}

func (self *Demuxer) poll() (err error) {
	if err = self.readTSPacket(); err == io.EOF {
		var n int
		if n, err = self.payloadEnd(); err != nil {
			return
		}
		if n == 0 {
			err = io.EOF
		}
	}
	return
}

func (self *Demuxer) initPMT(payload []byte) (err error) {
	var psihdrlen int
	var datalen int
	if _, _, psihdrlen, datalen, err = tsio.ParsePSI(payload); err != nil {
		return
	}
	self.pmt = &tsio.PMT{}
	if _, err = self.pmt.Unmarshal(payload[psihdrlen : psihdrlen+datalen]); err != nil {
		return
	}

	self.streams = []*Stream{}
	for i, info := range self.pmt.ElementaryStreamInfos {
		stream := &Stream{}
		stream.idx = i
		stream.demuxer = self
		stream.pid = info.ElementaryPID
		stream.streamType = info.StreamType
		if info.StreamType == 27 {
			stream.streamTypeDes = "h264"
		} else if info.StreamType == 15 {
			stream.streamTypeDes = "avc"
		} else {
			stream.streamTypeDes = "other"
		}
		switch info.StreamType {
		case tsio.ElementaryStreamTypeH264:
			self.streams = append(self.streams, stream)
		case tsio.ElementaryStreamTypeAdtsAAC:
			self.streams = append(self.streams, stream)
		}
	}
	return
}

func (self *Demuxer) payloadEnd() (n int, err error) {
	for _, stream := range self.streams {
		var i int
		if i, err = stream.payloadEnd(); err != nil {
			return
		}
		n += i
	}
	return
}

// find pat, pmt, and pes
func (self *Demuxer) readTSPacket() (err error) {
	var hdrlen int
	var pid uint16
	var start bool
	var iskeyframe bool

	if _, err = io.ReadFull(self.r, self.tshdr); err != nil {
		return
	}

	if pid, start, iskeyframe, hdrlen, err = tsio.ParseTSHeader(self.tshdr); err != nil {
		return
	}

	// fmt.Println("Got header")
	payload := self.tshdr[hdrlen:]

	if self.pat == nil {
		if pid == 0 {
			var psihdrlen int
			var datalen int
			if _, _, psihdrlen, datalen, err = tsio.ParsePSI(payload); err != nil {
				return
			}
			self.pat = &tsio.PAT{}
			if _, err = self.pat.Unmarshal(payload[psihdrlen : psihdrlen+datalen]); err != nil {
				return
			}
			// fmt.Println("Got pat")
			self.Pat = self.pat
		}
	} else if self.pmt == nil {
		for _, entry := range self.pat.Entries {
			if entry.ProgramMapPID == pid {
				if err = self.initPMT(payload); err != nil {
					return
				}
				// fmt.Println("Got pmt")
				self.Pmt = self.pmt
				break
			}
		}
	} else {
		for _, stream := range self.streams {
			if pid == stream.pid {
				if err = stream.handleTSPacket(start, iskeyframe, payload); err != nil {
					return
				}
				// fmt.Println("Got ts")
				break
			}
		}
	}

	return
}

func (self *Stream) addPacket(payload []byte, timedelta time.Duration) {
	dts := self.dts
	pts := self.pts
	if dts == 0 {
		dts = pts
	}

	demuxer := self.demuxer
	pkt := av.Packet{
		Idx:        int8(self.idx),
		IsKeyFrame: self.iskeyframe,
		Time:       dts + timedelta,
		Data:       payload,
	}
	if pts != dts {
		pkt.CompositionTime = pts - dts
	}
	demuxer.pkts = append(demuxer.pkts, pkt)
}

func (self *Stream) payloadEnd() (n int, err error) {
	payload := self.data
	if payload == nil {
		return
	}
	if self.datalen != 0 && len(payload) != self.datalen {
		err = fmt.Errorf("ts: packet size mismatch size=%d correct=%d", len(payload), self.datalen)
		return
	}
	self.data = nil

	switch self.streamType {
	case tsio.ElementaryStreamTypeAdtsAAC:
		self.demuxer.Payloads = append(self.demuxer.Payloads, payload)
		payload_info.PayloadInfo = payload
		self.demuxer.PayloadInfos = append(self.demuxer.PayloadInfos, payload_info)

		// self.demuxer.PayloadInfos.PayloadInfo = append(self.demuxer.PayloadInfos.PayloadInfo, payload)
		// fmt.Println("Payload aac end")
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
			n++
			delta += time.Duration(samples) * time.Second / time.Duration(config.SampleRate)
			payload = payload[framelen:]
		}

	case tsio.ElementaryStreamTypeH264:
		self.demuxer.Payloads = append(self.demuxer.Payloads, payload)
		// self.demuxer.PayloadInfos.PayloadInfo = append(self.demuxer.PayloadInfos.PayloadInfo, payload)
		payload_info.PayloadInfo = payload
		self.demuxer.PayloadInfos = append(self.demuxer.PayloadInfos, payload_info)

		// fmt.Println("Payload h264 end")
		// nalus, typ := h264parser.SplitNALUs(payload)
		nalus, _ := h264parser.SplitNALUs(payload)

		// spew.Dump(typ)
		var sps, pps []byte
		for _, nalu := range nalus {
			if len(nalu) > 0 {
				naltype := nalu[0] & 0x1f
				// spew.Dump(naltype)
				switch {
				case naltype == 7:
					sps = nalu
					// spew.Dump("got sps")
				case naltype == 8:
					pps = nalu
					// spew.Dump("got pps")
				case h264parser.IsDataNALU(nalu):
					// raw nalu to avcc
					b := make([]byte, 4+len(nalu))
					pio.PutU32BE(b[0:4], uint32(len(nalu)))
					copy(b[4:], nalu)
					self.addPacket(b, time.Duration(0))
					n++
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

var total int
var payload_info TPayloadInfos

func (self *Stream) handleTSPacket(start bool, iskeyframe bool, payload []byte) (err error) {
	if start {
		if _, err = self.payloadEnd(); err != nil {
			return
		}
		var hdrlen int
		if hdrlen, _, self.datalen, self.pts, self.dts, err = tsio.ParsePESHeader(payload); err != nil {
			return
		}
		payload_info.Pts = int(self.pts)
		payload_info.Dts = int(self.dts)
		self.iskeyframe = iskeyframe
		if self.datalen == 0 {
			self.data = make([]byte, 0, 4096)
		} else {
			self.data = make([]byte, 0, self.datalen)
		}
		self.data = append(self.data, payload[hdrlen:]...)
	} else {
		self.data = append(self.data, payload...)
	}
	total = total + 1
	// spew.Dump(total)
	return
}
