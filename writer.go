
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
	return self.writeSample(func (w io.Writer, data []byte) (int, error) {
		if _, err = self.mdatWriter.Write(data); err != nil {
			return 0, err
		}
		return len(data), nil
	}, sync, duration, data)
}

func (self *SimpleH264Writer) WriteNALU(sync bool, duration int, data []byte) (err error) {
	return self.writeSample(atom.WriteSampleByNALU, sync, duration, data)
}

func (self *SimpleH264Writer) writeSample(
	writeFunc func(io.Writer, []byte) (int,error),
	sync bool, duration int, data []byte,
) (err error) {
	if self.mdatWriter == nil {
		if err = self.prepare(); err != nil {
			return
		}
	}

	var sampleSize int
	if sampleSize, err = writeFunc(self.mdatWriter, data); err != nil {
		return
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
	}

	track := &atom.Track{
		Header: &atom.TrackHeader{
			Flags: 0x0001, // enabled
			Duration: self.duration,
			Volume: atom.IntToFixed(1),
			Matrix: [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
			TrackId: 1,
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

