
package mp4

import (
	"github.com/nareix/mp4/atom"
	"io"
)

const (
	H264 = 1
	AAC = 2
)

type Track struct {
	Type int
	SPS []byte
	PPS []byte
	TrackAtom *atom.Track
	r io.ReadSeeker

	sample *atom.SampleTable
	sampleIndex int

	ptsEntryIndex int
	sampleIndexInPtsEntry int

	dtsEntryIndex int
	sampleIndexInDtsEntry int

	chunkGroupIndex int
	chunkIndex int
	sampleIndexInChunk int
}

