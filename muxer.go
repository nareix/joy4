
package mp4

import (
	"github.com/nareix/mp4/atom"
	"github.com/nareix/mp4/isom"
	"io"
	"bytes"
	"fmt"
)

type Muxer struct {
	W io.WriteSeeker
	Tracks []*Track
	TrackH264 *Track
	TrackAAC *Track

	mdatWriter *atom.Writer
}

func (self *Muxer) newTrack() *Track {
	track := &Track{}

	track.sample = &atom.SampleTable{
		SampleDesc: &atom.SampleDesc{},
		TimeToSample: &atom.TimeToSample{},
		SampleToChunk: &atom.SampleToChunk{
			Entries: []atom.SampleToChunkEntry{
				{
					FirstChunk: 1,
					SampleDescId: 1,
					SamplesPerChunk: 1,
				},
			},
		},
		SampleSize: &atom.SampleSize{},
		ChunkOffset: &atom.ChunkOffset{},
	}

	track.TrackAtom = &atom.Track{
		Header: &atom.TrackHeader{
			TrackId: len(self.Tracks)+1,
			Flags: 0x0003, // Track enabled | Track in movie
			Duration: 0, // fill later
			Matrix: [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		},
		Media: &atom.Media{
			Header: &atom.MediaHeader{
				TimeScale: 0, // fill later
				Duration: 0, // fill later
				Language: 21956,
			},
			Info: &atom.MediaInfo{
				Sample: track.sample,
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

	track.writeMdat = self.writeMdat
	self.Tracks = append(self.Tracks, track)

	return track
}

func (self *Muxer) AddAACTrack() (track *Track) {
	track = self.newTrack()
	self.TrackAAC = track
	track.Type = AAC
	track.sample.SampleDesc.Mp4aDesc = &atom.Mp4aDesc{
		DataRefIdx: 1,
		NumberOfChannels: 0, // fill later
		SampleSize: 0, // fill later
		SampleRate: 0, // fill later
		Conf: &atom.ElemStreamDesc{},
	}
	track.TrackAtom.Header.Volume = atom.IntToFixed(1)
	track.TrackAtom.Header.AlternateGroup = 1
	track.TrackAtom.Media.Handler = &atom.HandlerRefer{
		SubType: "soun",
		Name: "Sound Handler",
	}
	track.TrackAtom.Media.Info.Sound = &atom.SoundMediaInfo{}
	return
}

func (self *Muxer) AddH264Track() (track *Track) {
	track = self.newTrack()
	self.TrackH264 = track
	track.Type = H264
	track.sample.SampleDesc.Avc1Desc = &atom.Avc1Desc{
		DataRefIdx: 1,
		HorizontalResolution: 72,
		VorizontalResolution: 72,
		Width: 0, // fill later
		Height: 0, // fill later
		FrameCount: 1,
		Depth: 24,
		ColorTableId: -1,
		Conf: &atom.Avc1Conf{},
	}
	track.sample.SyncSample = &atom.SyncSample{}
	track.TrackAtom.Media.Handler = &atom.HandlerRefer{
		SubType: "vide",
		Name: "Video Media Handler",
	}
	track.sample.CompositionOffset = &atom.CompositionOffset{}
	track.TrackAtom.Media.Info.Video = &atom.VideoMediaInfo{
		Flags: 0x000001,
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

func (self *Track) SetH264PPSAndSPS(pps, sps []byte) {
	self.pps, self.sps = pps, sps
}

func (self *Track) SetMPEG4AudioConfig(config isom.MPEG4AudioConfig) {
	self.mpeg4AudioConfig = config
}

func (self *Track) SetTimeScale(timeScale int64) {
	self.TrackAtom.Media.Header.TimeScale = int(timeScale)
	return
}

func (self *Track) WriteSample(pts int64, dts int64, isKeyFrame bool, data []byte) (err error) {
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
		duration := int(dts-self.lastDts)
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
		offset := int(pts-dts)
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

func (self *Track) fillTrackAtom() (err error) {
	if self.sampleIndex > 0 {
		self.sttsEntry.Count++
	}
	if self.Type == H264 {
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
		self.sample.SampleDesc.Avc1Desc.Width = int(info.Width)
		self.sample.SampleDesc.Avc1Desc.Height = int(info.Height)
		self.TrackAtom.Header.Duration = int(self.lastDts)
		self.TrackAtom.Header.TrackWidth = atom.IntToFixed(int(info.Width))
		self.TrackAtom.Header.TrackHeight = atom.IntToFixed(int(info.Height))
		self.TrackAtom.Media.Header.Duration = int(self.lastDts)
	} else if self.Type == AAC {
		buf := &bytes.Buffer{}
		config := self.mpeg4AudioConfig.Complete()
		if err = isom.WriteElemStreamDescAAC(buf, config, uint(self.TrackAtom.Header.TrackId)); err != nil {
			return
		}
		self.sample.SampleDesc.Mp4aDesc.Conf.Data = buf.Bytes()
		self.sample.SampleDesc.Mp4aDesc.NumberOfChannels = config.ChannelCount
		self.sample.SampleDesc.Mp4aDesc.SampleSize = config.ChannelCount*8
		self.sample.SampleDesc.Mp4aDesc.SampleRate = atom.IntToFixed(config.SampleRate)
		self.TrackAtom.Header.Duration = int(self.lastDts)
		self.TrackAtom.Media.Header.Duration = int(self.lastDts)
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {
	moov := &atom.Movie{}
	moov.Header = &atom.MovieHeader{
		PreferredRate: atom.IntToFixed(1),
		PreferredVolume: atom.IntToFixed(1),
		Matrix: [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		NextTrackId: 2,
	}
	timeScale := 0
	duration := 0
	for _, track := range(self.Tracks) {
		if err = track.fillTrackAtom(); err != nil {
			return
		}
		if track.TrackAtom.Media.Header.TimeScale > timeScale {
			timeScale = track.TrackAtom.Media.Header.TimeScale
		}
		if track.TrackAtom.Media.Header.Duration > duration {
			duration = track.TrackAtom.Media.Header.Duration
		}
		moov.Tracks = append(moov.Tracks, track.TrackAtom)
	}
	moov.Header.TimeScale = timeScale
	moov.Header.Duration = duration

	if err = self.mdatWriter.Close(); err != nil {
		return
	}
	if err = atom.WriteMovie(self.W, moov); err != nil {
		return
	}

	return
}

/*
type SimpleH264Writer struct {
	W io.WriteSeeker

	TimeScale int
	sps []byte
	pps []byte

	Width int
	Height int

	duration int

	sample *atom.SampleTable
	sampleToChunk *atom.SampleToChunkEntry
	sampleIdx int
	timeToSample *atom.TimeToSampleEntry

	mdatWriter *atom.Writer
}

func (self *SimpleH264Writer) prepare() (err error) {
	if self.mdatWriter, err = atom.WriteAtomHeader(self.W, "mdat"); err != nil {
		return
	}

	if len(self.sps) == 0 {
		err = fmt.Errorf("invalid sps")
		return
	}

	if len(self.pps) == 0 {
		err = fmt.Errorf("invalid pps")
		return
	}

	if self.Width == 0 || self.Height == 0 {
		var info *atom.H264spsInfo
		if info, err = atom.ParseH264sps(self.sps[1:]); err != nil {
			return
		}
		self.Width = int(info.Width)
		self.Height = int(info.Height)
	}

	self.sampleIdx = 1

	self.sample = &atom.SampleTable{
		SampleDesc: &atom.SampleDesc{
			Avc1Desc: &atom.Avc1Desc{
				DataRefIdx: 1,
				HorizontalResolution: 72,
				VorizontalResolution: 72,
				Width: self.Width,
				Height: self.Height,
				FrameCount: 1,
				Depth: 24,
				ColorTableId: -1,
				Conf: &atom.Avc1Conf{},
			},
		},
		TimeToSample: &atom.TimeToSample{},
		SampleToChunk: &atom.SampleToChunk{
			Entries: []atom.SampleToChunkEntry{
				{
					FirstChunk: 1,
					SampleDescId: 1,
				},
			},
		},
		SampleSize: &atom.SampleSize{},
		ChunkOffset: &atom.ChunkOffset{
			Entries: []int{8},
		},
		SyncSample: &atom.SyncSample{},
	}
	self.sampleToChunk = &self.sample.SampleToChunk.Entries[0]

	return
}

func (self *SimpleH264Writer) WriteSample(sync bool, duration int, data []byte) (err error) {
	return self.writeSample(false, sync, duration, data)
}

func (self *SimpleH264Writer) WriteNALU(sync bool, duration int, data []byte) (err error) {
	return self.writeSample(true, sync, duration, data)
}

func splitNALUByStartCode(data []byte) (out [][]byte) {
	last := 0
	for i := 0; i < len(data)-3; {
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 {
			out = append(out, data[last:i])
			i += 3
			last = i
		} else {
			i++
		}
	}
	out = append(out, data[last:])
	return
}

func (self *SimpleH264Writer) writeSample(isNALU, sync bool, duration int, data []byte) (err error) {
	if self.mdatWriter == nil {
		if err = self.prepare(); err != nil {
			return
		}
	}

	var sampleSize int

	if isNALU {
		if sampleSize, err = atom.WriteSampleByNALU(self.mdatWriter, data); err != nil {
			return
		}
	} else {
		sampleSize = len(data)
		if _, err = self.mdatWriter.Write(data); err != nil {
			return
		}
	}

	if sync {
		self.sample.SyncSample.Entries = append(self.sample.SyncSample.Entries, self.sampleIdx)
	}

	if self.timeToSample != nil && duration != self.timeToSample.Duration {
		self.sample.TimeToSample.Entries = append(self.sample.TimeToSample.Entries, *self.timeToSample)
		self.timeToSample = nil
	}
	if self.timeToSample == nil {
		self.timeToSample = &atom.TimeToSampleEntry{
			Duration: duration,
		}
	}

	self.duration += duration
	self.sampleIdx++
	self.timeToSample.Count++
	self.sampleToChunk.SamplesPerChunk++
	self.sample.SampleSize.Entries = append(self.sample.SampleSize.Entries, sampleSize)

	return
}

func (self *SimpleH264Writer) Finish() (err error) {
	self.sample.SampleDesc.Avc1Desc.Conf.Record, err = atom.CreateAVCDecoderConfRecord(
		self.sps,
		self.pps,
	)
	if err != nil {
		return
	}

	if self.timeToSample != nil {
		self.sample.TimeToSample.Entries = append(
			self.sample.TimeToSample.Entries,
			*self.timeToSample,
		)
	}

	if err = self.mdatWriter.Close(); err != nil {
		return
	}

	moov := &atom.Movie{}
	moov.Header = &atom.MovieHeader{
		TimeScale: self.TimeScale,
		Duration: self.duration,
		PreferredRate: atom.IntToFixed(1),
		PreferredVolume: atom.IntToFixed(1),
		Matrix: [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		NextTrackId: 2,
	}

	track := &atom.Track{
		Header: &atom.TrackHeader{
			TrackId: 1,
			Flags: 0x0003, // Track enabled | Track in movie
			Duration: self.duration,
			Volume: atom.IntToFixed(1),
			Matrix: [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
			TrackWidth: atom.IntToFixed(self.Width),
			TrackHeight: atom.IntToFixed(self.Height),
		},

		Media: &atom.Media{
			Header: &atom.MediaHeader{
				TimeScale: self.TimeScale,
				Duration: self.duration,
			},
			Info: &atom.MediaInfo{
				Video: &atom.VideoMediaInfo{
					Flags: 0x000001,
				},
				Sample: self.sample,
				Data: &atom.DataInfo{
					Refer: &atom.DataRefer{
						Url: &atom.DataReferUrl{
							Flags: 0x000001, // Self reference
						},
					},
				},
			},
			Handler: &atom.HandlerRefer{
				SubType: "vide",
				Name: "Video Media Handler",
			},
		},
	}
	moov.Tracks = append(moov.Tracks, track)

	if err = atom.WriteMovie(self.W, moov); err != nil {
		return
	}

	return
}
*/

