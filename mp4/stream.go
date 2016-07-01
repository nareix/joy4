package mp4

import (
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/mp4/atom"
	"time"
	"io"
)

type Stream struct {
	av.CodecData

	trackAtom *atom.Track
	r         io.ReadSeeker
	idx       int

	lasttime time.Duration

	timeScale int64
	duration  int64

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
}

func timeToTs(tm time.Duration, timeScale int64) int64 {
	return int64(tm*time.Duration(timeScale) / time.Second)
}

func tsToTime(ts int64, timeScale int64) time.Duration {
	return time.Duration(ts)*time.Second / time.Duration(timeScale)
}

func (self *Stream) timeToTs(tm time.Duration) int64 {
	return int64(tm*time.Duration(self.timeScale) / time.Second)
}

func (self *Stream) tsToTime(ts int64) time.Duration {
	return time.Duration(ts)*time.Second / time.Duration(self.timeScale)
}
