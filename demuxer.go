package ts

import (
	"bytes"
	"fmt"
	"github.com/nareix/codec/aacparser"
	"io"
)

type Demuxer struct {
	R io.Reader

	pat       PAT
	pmt       *PMT
	Tracks    []*Stream
	TrackH264 *Stream
	TrackAAC  *Stream
}

// ParsePacket() (pid uint, counter int, isStart bool, pts, dst int64, isKeyFrame bool)
// WritePayload(pid, pts, dts, isKeyFrame, payloads, isVideoFrame)

func (self *Demuxer) TimeScale() int64 {
	return PTS_HZ
}

func (self *Demuxer) ReadHeader() (err error) {
	self.Tracks = []*Stream{}
	self.TrackH264 = nil
	self.TrackAAC = nil

	for {
		if self.pmt != nil {
			n := 0
			for _, stream := range self.Tracks {
				if stream.payloadReady {
					n++
				}
			}
			if n == len(self.Tracks) {
				break
			}
		}

		if err = self.readPacket(); err != nil {
			return
		}
	}

	return
}

func (self *Demuxer) ReadSample() (stream *Stream, err error) {
	if len(self.Tracks) == 0 {
		err = fmt.Errorf("no track")
		return
	}

	for {
		for _, _track := range self.Tracks {
			if _track.payloadReady {
				stream = _track
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
							stream.Type = H264
							self.TrackH264 = stream
							self.Tracks = append(self.Tracks, stream)
						case ElementaryStreamTypeAdtsAAC:
							stream.Type = AAC
							self.TrackAAC = stream
							self.Tracks = append(self.Tracks, stream)
						}
					}
				}
			}
		} else {

			for _, stream := range self.Tracks {
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

func (self *Stream) GetMPEG4AudioConfig() aacparser.MPEG4AudioConfig {
	return self.mpeg4AudioConfig
}

func (self *Stream) ReadSample() (pts int64, dts int64, isKeyFrame bool, data []byte, err error) {
	for !self.payloadReady {
		if err = self.demuxer.readPacket(); err != nil {
			return
		}
	}

	dts = int64(self.peshdr.DTS)
	pts = int64(self.peshdr.PTS)
	if dts == 0 {
		dts = pts
	}
	isKeyFrame = self.tshdr.RandomAccessIndicator
	data = self.payload
	self.payloadReady = false

	return
}

func (self *Stream) appendPayload() (err error) {
	self.payload = self.buf.Bytes()

	if self.Type == AAC {
		if !self.mpeg4AudioConfig.IsValid() {
			if self.mpeg4AudioConfig, _, _, _, err = aacparser.ReadADTSFrame(self.payload); err != nil {
				return
			}
			self.mpeg4AudioConfig = self.mpeg4AudioConfig.Complete()
			if !self.mpeg4AudioConfig.IsValid() {
				err = fmt.Errorf("invalid MPEG4AudioConfig")
				return
			}
		}
	}

	self.payloadReady = true
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
		self.payloadReady = false
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
