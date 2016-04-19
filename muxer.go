package mp4

import (
	"bytes"
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/mp4/atom"
	"github.com/nareix/mp4/isom"
	"io"
)

type Muxer struct {
	W          io.WriteSeeker
	streams    []*Stream
	mdatWriter *atom.Writer
}

func (self *Muxer) NewStream() av.Stream {
	stream := &Stream{}

	stream.sample = &atom.SampleTable{
		SampleDesc:   &atom.SampleDesc{},
		TimeToSample: &atom.TimeToSample{},
		SampleToChunk: &atom.SampleToChunk{
			Entries: []atom.SampleToChunkEntry{
				{
					FirstChunk:      1,
					SampleDescId:    1,
					SamplesPerChunk: 1,
				},
			},
		},
		SampleSize:  &atom.SampleSize{},
		ChunkOffset: &atom.ChunkOffset{},
	}

	stream.trackAtom = &atom.Track{
		Header: &atom.TrackHeader{
			TrackId:  len(self.streams) + 1,
			Flags:    0x0003, // Track enabled | Track in movie
			Duration: 0,      // fill later
			Matrix:   [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		},
		Media: &atom.Media{
			Header: &atom.MediaHeader{
				TimeScale: 0, // fill later
				Duration:  0, // fill later
				Language:  21956,
			},
			Info: &atom.MediaInfo{
				Sample: stream.sample,
				Data: &atom.DataInfo{
					Refer: &atom.DataRefer{
						Url: &atom.DataReferUrl{
							Flags: 0x000001, // Self reference
						},
					},
				},
			},
		},
	}

	stream.writeMdat = self.writeMdat
	self.streams = append(self.streams, stream)

	return stream
}

func (self *Stream) fillTrackAtom() (err error) {
	if self.sampleIndex > 0 {
		self.sttsEntry.Count++
	}

	self.trackAtom.Media.Header.TimeScale = int(self.TimeScale())
	self.trackAtom.Media.Header.Duration = int(self.lastDts)

	if self.Type() == av.H264 {
		self.sample.SampleDesc.Avc1Desc.Conf.Record, err = atom.CreateAVCDecoderConfRecord(
			self.sps,
			self.pps,
		)
		if err != nil {
			return
		}
		var info *atom.H264SPSInfo
		if info, err = atom.ParseH264SPS(self.sps[1:]); err != nil {
			return
		}
		self.sample.SampleDesc.Avc1Desc = &atom.Avc1Desc{
			DataRefIdx:           1,
			HorizontalResolution: 72,
			VorizontalResolution: 72,
			Width:                int(info.Width),
			Height:               int(info.Height),
			FrameCount:           1,
			Depth:                24,
			ColorTableId:         -1,
			Conf:                 &atom.Avc1Conf{},
		}
		self.sample.SyncSample = &atom.SyncSample{}
		self.trackAtom.Media.Handler = &atom.HandlerRefer{
			SubType: "vide",
			Name:    "Video Media Handler",
		}
		self.sample.CompositionOffset = &atom.CompositionOffset{}
		self.trackAtom.Media.Info.Video = &atom.VideoMediaInfo{
			Flags: 0x000001,
		}
		self.trackAtom.Header.TrackWidth = atom.IntToFixed(int(info.Width))
		self.trackAtom.Header.TrackHeight = atom.IntToFixed(int(info.Height))

	} else if self.Type() == av.AAC {
		if !self.mpeg4AudioConfig.IsValid() {
			err = fmt.Errorf("invalie MPEG4AudioConfig")
			return
		}
		buf := &bytes.Buffer{}
		config := self.mpeg4AudioConfig.Complete()
		if err = isom.WriteElemStreamDescAAC(buf, config, uint(self.trackAtom.Header.TrackId)); err != nil {
			return
		}
		self.sample.SampleDesc.Mp4aDesc = &atom.Mp4aDesc{
			DataRefIdx:       1,
			NumberOfChannels: config.ChannelCount,
			SampleSize:       config.ChannelCount * 8,
			SampleRate:       atom.IntToFixed(config.SampleRate),
			Conf: &atom.ElemStreamDesc{
				Data: buf.Bytes(),
			},
		}
		self.trackAtom.Header.Volume = atom.IntToFixed(1)
		self.trackAtom.Header.AlternateGroup = 1
		self.trackAtom.Media.Handler = &atom.HandlerRefer{
			SubType: "soun",
			Name:    "Sound Handler",
		}
		self.trackAtom.Media.Info.Sound = &atom.SoundMediaInfo{}

	} else {
		err = fmt.Errorf("please specify stream type")
	}

	return
}

func (self *Muxer) writeMdat(data []byte) (pos int64, err error) {
	if pos, err = self.mdatWriter.Seek(0, 1); err != nil {
		return
	}
	_, err = self.mdatWriter.Write(data)
	return
}

func (self *Muxer) WriteHeader() (err error) {
	if self.mdatWriter, err = atom.WriteAtomHeader(self.W, "mdat"); err != nil {
		return
	}
	return
}

func (self *Muxer) WriteSample(pkt av.Packet) (err error) {
	pts, dts, isKeyFrame, frame := pkt.Pts, pkt.Dts, pkt.IsKeyFrame, pkt.Data
	stream := self.streams[pkt.StreamIdx]

	if stream.Type() == av.AAC && isom.IsADTSFrame(frame) {
		config := stream.mpeg4AudioConfig.Complete()
		if config.SampleRate == 0 {
			err = fmt.Errorf("invalid sample rate")
			return
		}
		for len(frame) > 0 {
			var payload []byte
			var samples int
			var framelen int
			if _, payload, samples, framelen, err = isom.ReadADTSFrame(frame); err != nil {
				return
			}
			delta := int64(samples) * stream.TimeScale() / int64(config.SampleRate)
			pts += delta
			dts += delta
			frame = frame[framelen:]
			if stream.writeSample(pts, dts, isKeyFrame, payload); err != nil {
				return
			}
		}
	}

	return stream.writeSample(pts, dts, isKeyFrame, frame)
}

func (self *Stream) writeSample(pts int64, dts int64, isKeyFrame bool, data []byte) (err error) {
	var filePos int64
	sampleSize := len(data)
	if filePos, err = self.writeMdat(data); err != nil {
		return
	}

	if isKeyFrame && self.sample.SyncSample != nil {
		self.sample.SyncSample.Entries = append(self.sample.SyncSample.Entries, self.sampleIndex+1)
	}

	if self.sampleIndex > 0 {
		if dts <= self.lastDts {
			err = fmt.Errorf("dts must be incremental")
			return
		}
		duration := int(dts - self.lastDts)
		if self.sttsEntry == nil || duration != self.sttsEntry.Duration {
			self.sample.TimeToSample.Entries = append(self.sample.TimeToSample.Entries, atom.TimeToSampleEntry{Duration: duration})
			self.sttsEntry = &self.sample.TimeToSample.Entries[len(self.sample.TimeToSample.Entries)-1]
		}
		self.sttsEntry.Count++
	}

	if self.sample.CompositionOffset != nil {
		if pts < dts {
			err = fmt.Errorf("pts must greater than dts")
			return
		}
		offset := int(pts - dts)
		if self.cttsEntry == nil || offset != self.cttsEntry.Offset {
			table := self.sample.CompositionOffset
			table.Entries = append(table.Entries, atom.CompositionOffsetEntry{Offset: offset})
			self.cttsEntry = &table.Entries[len(table.Entries)-1]
		}
		self.cttsEntry.Count++
	}

	self.lastDts = dts
	self.sampleIndex++
	self.sample.ChunkOffset.Entries = append(self.sample.ChunkOffset.Entries, int(filePos))
	self.sample.SampleSize.Entries = append(self.sample.SampleSize.Entries, sampleSize)

	return
}

func (self *Muxer) WriteTrailer() (err error) {
	moov := &atom.Movie{}
	moov.Header = &atom.MovieHeader{
		PreferredRate:   atom.IntToFixed(1),
		PreferredVolume: atom.IntToFixed(1),
		Matrix:          [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		NextTrackId:     2,
	}

	maxDur := float64(0)
	timeScale := 10000
	for _, stream := range self.streams {
		if err = stream.fillTrackAtom(); err != nil {
			return
		}
		dur := stream.Duration()
		stream.trackAtom.Header.Duration = int(float64(timeScale) * dur)
		if dur > maxDur {
			maxDur = dur
		}
		moov.Tracks = append(moov.Tracks, stream.trackAtom)
	}
	moov.Header.TimeScale = timeScale
	moov.Header.Duration = int(float64(timeScale) * maxDur)

	if err = self.mdatWriter.Close(); err != nil {
		return
	}
	if err = atom.WriteMovie(self.W, moov); err != nil {
		return
	}

	return
}
