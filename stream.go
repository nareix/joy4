package mp4

import (
	"github.com/nareix/av"
	"github.com/nareix/mp4/atom"
	"io"
)

type Stream struct {
	av.StreamCommon

	trackAtom *atom.Track
	r         io.ReadSeeker
	idx       int

	timeScale int64
	duration int64

	muxer *Muxer

	sample      *atom.SampleTable
	sampleIndex int

	sampleOffsetInChunk int64
	syncSampleIndex     int

	dts                    int64
	sttsEntryIndex         int
	sampleIndexInSttsEntry int

	cttsEntryIndex         int
	sampleIndexInCttsEntry int

	chunkGroupIndex    int
	chunkIndex         int
	sampleIndexInChunk int

	sttsEntry *atom.TimeToSampleEntry
	cttsEntry *atom.CompositionOffsetEntry
	writeMdat func([]byte) (int64, error)
}

func (self *Stream) timeToTs(time float64) int64 {
	return int64(time * float64(self.timeScale))
}

func (self *Stream) tsToTime(ts int64) float64 {
	return float64(ts) / float64(self.timeScale)
}
