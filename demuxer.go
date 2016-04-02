
package ts

import (
	"io"
	"bytes"
	"fmt"
	"github.com/nareix/mp4/isom"
)

type Demuxer struct {
	R io.Reader

	pat PAT
	pmt *PMT
	Tracks []*Track
	TrackH264 *Track
	TrackAAC *Track
}

// ParsePacket() (pid uint, counter int, isStart bool, pts, dst int64, isKeyFrame bool)
// WritePayload(pid, pts, dts, isKeyFrame, payloads, isVideoFrame)

func (self *Demuxer) TimeScale() int64 {
	return PTS_HZ
}

func (self *Demuxer) ReadHeader() (err error) {
	self.Tracks = []*Track{}
	self.TrackH264 = nil
	self.TrackAAC = nil

	for {
		if self.pmt != nil {
			n := 0
			for _, track := range(self.Tracks) {
				if track.payloadReady {
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

func (self *Demuxer) ReadSample() (track *Track, err error) {
	if len(self.Tracks) == 0 {
		err = fmt.Errorf("no track")
		return
	}

	for {
		for _, _track := range(self.Tracks) {
			if _track.payloadReady {
				track = _track
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
			for _, entry := range(self.pat.Entries) {
				if entry.ProgramMapPID == header.PID {
					self.pmt = new(PMT)
					if *self.pmt, err = ReadPMT(bytes.NewReader(payload)); err != nil {
						return
					}
					for _, info := range(self.pmt.ElementaryStreamInfos) {
						track := &Track{}

						track.demuxer = self
						track.pid = info.ElementaryPID
						switch info.StreamType {
						case ElementaryStreamTypeH264:
							track.Type = H264
							self.TrackH264 = track
							self.Tracks = append(self.Tracks, track)
						case ElementaryStreamTypeAdtsAAC:
							track.Type = AAC
							self.TrackAAC = track
							self.Tracks = append(self.Tracks, track)
						}
					}
				}
			}
		} else {

			for _, track := range(self.Tracks) {
				if header.PID == track.pid {
					if err = track.appendPacket(header, payload); err != nil {
						return
					}
				}
			}
		}
	}

	return
}

func (self *Track) GetMPEG4AudioConfig() isom.MPEG4AudioConfig {
	return self.mpeg4AudioConfig
}

func (self *Track) ReadSample() (pts int64, dts int64, isKeyFrame bool, data []byte, err error) {
	for !self.payloadReady {
		if err = self.demuxer.readPacket(); err != nil {
			return
		}
	}

	if self.Type == AAC {
		var n int
		if _, data, n, self.payload, err = isom.ReadADTSPayload(self.payload); err != nil {
			return
		}
		pts = self.PTS
		dts = pts
		self.PTS += int64(PTS_HZ*n)/int64(self.mpeg4AudioConfig.SampleRate)
		if len(self.payload) == 0 {
			self.payloadReady = false
		}
	} else {
		dts = int64(self.peshdr.DTS)
		pts = int64(self.peshdr.PTS)
		isKeyFrame = self.tshdr.RandomAccessIndicator
		data = self.payload
		self.payloadReady = false
	}

	if dts == 0 {
		dts = pts
	}
	return
}

func (self *Track) appendPayload() (err error) {
	self.payload = self.buf.Bytes()
	if len(self.payload) == 0 {
		err = fmt.Errorf("empty payload")
		return
	}

	if self.Type == AAC {
		if !self.mpeg4AudioConfig.IsValid() {
			if self.mpeg4AudioConfig, _, _, _, err = isom.ReadADTSPayload(self.payload); err != nil {
				return
			}
			self.mpeg4AudioConfig = self.mpeg4AudioConfig.Complete()
			if !self.mpeg4AudioConfig.IsValid() {
				err = fmt.Errorf("invalid MPEG4AudioConfig")
				return
			}
		}
		self.PTS = int64(self.peshdr.PTS)
	}

	self.payloadReady = true
	return
}

func (self *Track) appendPacket(header TSHeader, payload []byte) (err error) {
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

