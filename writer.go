
package mp4

import (
	"github.com/nareix/mp4/atom"
	"io"
)

type SimpleH264Writer struct {
	W io.WriteSeeker

	TimeScale int
	SPS []byte
	PPS []byte

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
		self.SPS,
		self.PPS,
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

