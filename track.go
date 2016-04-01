
package mp4

import (
	"github.com/nareix/mp4/atom"
	"github.com/nareix/mp4/isom"
	"io"
)

const (
	H264 = 1
	AAC = 2
)

type Track struct {
	Type int
	TrackAtom *atom.Track
	r io.ReadSeeker

	sps []byte
	pps []byte

	mpeg4AudioConfig isom.MPEG4AudioConfig

	sample *atom.SampleTable
	sampleIndex int

	sampleOffsetInChunk int64
	syncSampleIndex int

	dts int64
	sttsEntryIndex int
	sampleIndexInSttsEntry int

	cttsEntryIndex int
	sampleIndexInCttsEntry int

	chunkGroupIndex int
	chunkIndex int
	sampleIndexInChunk int

	sttsEntry *atom.TimeToSampleEntry
	cttsEntry *atom.CompositionOffsetEntry
	writeMdat func ([]byte) (int64,error)
	lastDts int64
}

