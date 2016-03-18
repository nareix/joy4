
package mp4

import (
	"github.com/nareix/mp4/atom"
	_ "os"
	"fmt"
	"io"
	_ "log"
)

type Demuxer struct {
	R io.ReadSeeker
	Tracks []*Track
	TrackH264 *Track
	TrackAAC *Track
	MovieAtom *atom.Movie
}

func (self *Demuxer) ReadHeader() (err error) {
	var N int64
	var moov *atom.Movie

	if N, err = self.R.Seek(0, 2); err != nil {
		return
	}
	if _, err = self.R.Seek(0, 0); err != nil {
		return
	}

	lr := &io.LimitedReader{R: self.R, N: N}
	for lr.N > 0 {
		var ar *io.LimitedReader

		var cc4 string
		if ar, cc4, err = atom.ReadAtomHeader(lr, ""); err != nil {
			return
		}

		if cc4 == "moov" {
			if moov, err = atom.ReadMovie(ar); err != nil {
				return
			}
		}

		if _, err = atom.ReadDummy(lr, int(ar.N)); err != nil {
			return
		}
	}

	if moov == nil {
		err = fmt.Errorf("'moov' atom not found")
		return
	}
	self.MovieAtom = moov

	self.Tracks = []*Track{}
	for _, atrack := range(moov.Tracks) {
		track := &Track{
			TrackAtom: atrack,
			r: self.R,
		}
		if atrack.Media != nil && atrack.Media.Info != nil && atrack.Media.Info.Sample != nil {
			track.sample = atrack.Media.Info.Sample
		} else {
			err = fmt.Errorf("sample table not found")
			return
		}
		if record := atom.GetAVCDecoderConfRecordByTrack(atrack); record != nil {
			track.Type = H264
			self.TrackH264 = track
			if len(record.PPS) > 0 {
				track.PPS = record.PPS[0]
			}
			if len(record.SPS) > 0 {
				track.SPS = record.SPS[0]
			}
			self.Tracks = append(self.Tracks, track)
		} else if mp4a := atom.GetMp4aDescByTrack(atrack); mp4a != nil {
			self.TrackAAC = track
			track.Type = AAC
			self.Tracks = append(self.Tracks, track)
		}
	}

	return
}

func (self *Track) setSampleIndex(index int) (err error) {
	found := false
	start := 0
	self.chunkGroupIndex = 0

	for self.chunkIndex = range(self.sample.ChunkOffset.Entries) {
		n := self.sample.SampleToChunk.Entries[self.chunkGroupIndex].SamplesPerChunk
		if index >= start && index < start+n {
			found = true
			self.sampleIndexInChunk = index-start
			break
		}
		start += n
		if self.chunkGroupIndex+1 < len(self.sample.SampleToChunk.Entries) &&
			self.chunkIndex+1 == self.sample.SampleToChunk.Entries[self.chunkGroupIndex+1].FirstChunk {
			self.chunkGroupIndex++
		}
	}
	if !found {
		err = io.EOF
		return
	}

	start = 0
	found = false
	self.ptsEntryIndex = 0
	for self.ptsEntryIndex < len(self.sample.TimeToSample.Entries) {
		n := self.sample.TimeToSample.Entries[self.ptsEntryIndex].Count
		if index >= start && index < start+n {
			self.sampleIndexInPtsEntry = index-start
			break
		}
		start += n
		self.ptsEntryIndex++
	}
	if !found {
		err = io.EOF
		return
	}

	start = 0
	found = false
	self.dtsEntryIndex = 0
	for self.dtsEntryIndex < len(self.sample.CompositionOffset.Entries) {
		n := self.sample.CompositionOffset.Entries[self.dtsEntryIndex].Count
		if index >= start && index < start+n {
			self.sampleIndexInDtsEntry = index-start
			break
		}
		start += n
		self.dtsEntryIndex++
	}
	if !found {
		err = io.EOF
		return
	}

	self.sampleIndex = index
	return
}

func (self *Track) incSampleIndex() {
	self.sampleIndexInChunk++
	if self.sampleIndexInChunk == self.sample.SampleToChunk.Entries[self.chunkGroupIndex].SamplesPerChunk {
		self.chunkIndex++
		self.sampleIndexInChunk = 0
	}
	if self.chunkGroupIndex+1 < len(self.sample.SampleToChunk.Entries) &&
		self.chunkIndex+1 == self.sample.SampleToChunk.Entries[self.chunkGroupIndex+1].FirstChunk {
		self.chunkGroupIndex++
	}
	self.sampleIndex++
}

func (self *Track) SampleCount() int {
	if self.sample.SampleSize.SampleSize == 0 {
		chunkGroupIndex := 0
		count := 0
		for chunkIndex := range(self.sample.ChunkOffset.Entries) {
			n := self.sample.SampleToChunk.Entries[chunkGroupIndex].SamplesPerChunk
			count += n
			if chunkGroupIndex+1 < len(self.sample.SampleToChunk.Entries) &&
				chunkIndex+1 == self.sample.SampleToChunk.Entries[chunkGroupIndex+1].FirstChunk {
				chunkGroupIndex++
			}
		}
		return count
	} else {
		return len(self.sample.SampleSize.Entries)
	}
}

func (self *Track) ReadSample() (pts int64, dts int64, isKeyFrame bool, data []byte, err error) {
	return
}

func (self *Track) ReadSampleAtIndex(index int) (pts int64, dts int64, isKeyFrame bool, data []byte, err error) {
	if self.sampleIndex+1 == index {
		self.incSampleIndex()
	} else if self.sampleIndex != index {
		if err = self.setSampleIndex(index); err != nil {
			return
		}
	}

	if self.chunkIndex > len(self.sample.ChunkOffset.Entries) {
		err = io.EOF
		return
	}
	if self.chunkGroupIndex >= len(self.sample.SampleToChunk.Entries) {
		err = io.EOF
		return
	}

	chunkOffset := self.sample.ChunkOffset.Entries[self.chunkIndex]
	sampleOffset := 0
	sampleSize := 0

	if self.sample.SampleSize.SampleSize != 0 {
		sampleOffset = chunkOffset + self.sampleIndexInChunk*self.sample.SampleSize.SampleSize
	} else {
		sampleOffset = chunkOffset
		for i := self.sampleIndex-self.sampleIndexInChunk; i < self.sampleIndex; i++ {
			sampleOffset += self.sample.SampleSize.Entries[i]
		}
	}

	if _, err = self.r.Seek(int64(sampleOffset), 0); err != nil {
		return
	}
	data = make([]byte, sampleSize)
	if _, err = self.r.Read(data); err != nil {
		return
	}

	return
}

func (self *Track) Duration() float32 {
	total := int64(0)
	for _, entry := range(self.sample.TimeToSample.Entries) {
		total += int64(entry.Duration*entry.Count)
	}
	return float32(total)/float32(self.TrackAtom.Media.Header.TimeScale)
}

func (self *Track) TimeToSampleIndex(second float32) int {
	return 0
}

func (self *Track) TimeStampToTime(ts int64) float32 {
	return 0.0
}

func (self *Track) WriteSample(pts int64, dts int64, data []byte) (err error) {
	return
}

