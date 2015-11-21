// THIS FILE IS AUTO GENERATED
package atom

import (
	"io"
)

type FileType struct {
}

func ReadFileType(r *io.LimitedReader) (res *FileType, err error) {

	self := &FileType{}

	res = self
	return
}

type Movie struct {
	Header *MovieHeader
	Tracks []*Track
}

func ReadMovie(r *io.LimitedReader) (res *Movie, err error) {

	self := &Movie{}
	// ReadAtoms
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "mvhd":
			{
				if self.Header, err = ReadMovieHeader(ar); err != nil {
					return
				}
			}
		case "trak":
			{
				var item *Track
				if item, err = ReadTrack(ar); err != nil {
					return
				}
				self.Tracks = append(self.Tracks, item)
			}

		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}

type MovieHeader struct {
	Version           int
	Flags             int
	CTime             TimeStamp
	MTime             TimeStamp
	TimeScale         TimeStamp
	Duration          TimeStamp
	PreferredRate     int
	PreferredVolume   int
	Matrix            [9]int
	PreviewTime       TimeStamp
	PreviewDuration   TimeStamp
	PosterTime        TimeStamp
	SelectionTime     TimeStamp
	SelectionDuration TimeStamp
	CurrentTime       TimeStamp
	NextTrackId       int
}

func ReadMovieHeader(r *io.LimitedReader) (res *MovieHeader, err error) {

	self := &MovieHeader{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.CTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.MTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.TimeScale, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.PreferredRate, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.PreferredVolume, err = ReadInt(r, 2); err != nil {
		return
	}
	if _, err = ReadDummy(r, 10); err != nil {
		return
	}
	for i := 0; i < 9; i++ {
		if self.Matrix[i], err = ReadInt(r, 4); err != nil {
			return
		}
	}
	if self.PreviewTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.PreviewDuration, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.PosterTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.SelectionTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.SelectionDuration, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.CurrentTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.NextTrackId, err = ReadInt(r, 4); err != nil {
		return
	}
	res = self
	return
}

type Track struct {
	Header *TrackHeader
	Media  *Media
}

func ReadTrack(r *io.LimitedReader) (res *Track, err error) {

	self := &Track{}
	// ReadAtoms
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "tkhd":
			{
				if self.Header, err = ReadTrackHeader(ar); err != nil {
					return
				}
			}
		case "mdia":
			{
				if self.Media, err = ReadMedia(ar); err != nil {
					return
				}
			}

		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}

type TrackHeader struct {
	Version        int
	Flags          int
	CTime          TimeStamp
	MTime          TimeStamp
	TrackId        TimeStamp
	Duration       TimeStamp
	Layer          int
	AlternateGroup int
	Volume         int
	Matrix         [9]int
	TrackWidth     int
	TrackHeader    int
}

func ReadTrackHeader(r *io.LimitedReader) (res *TrackHeader, err error) {

	self := &TrackHeader{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.CTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.MTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.TrackId, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if _, err = ReadDummy(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if _, err = ReadDummy(r, 8); err != nil {
		return
	}
	if self.Layer, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.AlternateGroup, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Volume, err = ReadInt(r, 2); err != nil {
		return
	}
	if _, err = ReadDummy(r, 2); err != nil {
		return
	}
	for i := 0; i < 9; i++ {
		if self.Matrix[i], err = ReadInt(r, 4); err != nil {
			return
		}
	}
	if self.TrackWidth, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.TrackHeader, err = ReadInt(r, 4); err != nil {
		return
	}
	res = self
	return
}

type Media struct {
	Header *MediaHeader
	Info   *MediaInfo
}

func ReadMedia(r *io.LimitedReader) (res *Media, err error) {

	self := &Media{}
	// ReadAtoms
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "mdhd":
			{
				if self.Header, err = ReadMediaHeader(ar); err != nil {
					return
				}
			}
		case "minf":
			{
				if self.Info, err = ReadMediaInfo(ar); err != nil {
					return
				}
			}

		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}

type MediaHeader struct {
	Version   int
	Flags     int
	CTime     int
	MTime     int
	TimeScale int
	Duration  int
	Language  int
	Quality   int
}

func ReadMediaHeader(r *io.LimitedReader) (res *MediaHeader, err error) {

	self := &MediaHeader{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.CTime, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.MTime, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.TimeScale, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Language, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Quality, err = ReadInt(r, 2); err != nil {
		return
	}
	res = self
	return
}

type MediaInfo struct {
	Sound  *SoundMediaInfo
	Video  *VideoMediaInfo
	Sample *SampleTable
}

func ReadMediaInfo(r *io.LimitedReader) (res *MediaInfo, err error) {

	self := &MediaInfo{}
	// ReadAtoms
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "smhd":
			{
				if self.Sound, err = ReadSoundMediaInfo(ar); err != nil {
					return
				}
			}
		case "vmhd":
			{
				if self.Video, err = ReadVideoMediaInfo(ar); err != nil {
					return
				}
			}
		case "stbl":
			{
				if self.Sample, err = ReadSampleTable(ar); err != nil {
					return
				}
			}

		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}

type SoundMediaInfo struct {
	Version int
	Flags   int
	Balance int
}

func ReadSoundMediaInfo(r *io.LimitedReader) (res *SoundMediaInfo, err error) {

	self := &SoundMediaInfo{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.Balance, err = ReadInt(r, 2); err != nil {
		return
	}
	if _, err = ReadDummy(r, 2); err != nil {
		return
	}
	res = self
	return
}

type VideoMediaInfo struct {
	Version      int
	Flags        int
	GraphicsMode int
	Opcolor      [3]int
}

func ReadVideoMediaInfo(r *io.LimitedReader) (res *VideoMediaInfo, err error) {

	self := &VideoMediaInfo{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.GraphicsMode, err = ReadInt(r, 2); err != nil {
		return
	}
	for i := 0; i < 3; i++ {
		if self.Opcolor[i], err = ReadInt(r, 2); err != nil {
			return
		}
	}
	res = self
	return
}

type SampleTable struct {
	SampleDesc        *SampleDesc
	TimeToSample      *TimeToSample
	CompositionOffset *CompositionOffset
	SampleToChunk     *SampleToChunk
	SyncSample        *SyncSample
	ChunkOffset       *ChunkOffset
	SampleSize        *SampleSize
}

func ReadSampleTable(r *io.LimitedReader) (res *SampleTable, err error) {

	self := &SampleTable{}
	// ReadAtoms
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "stsd":
			{
				if self.SampleDesc, err = ReadSampleDesc(ar); err != nil {
					return
				}
			}
		case "stts":
			{
				if self.TimeToSample, err = ReadTimeToSample(ar); err != nil {
					return
				}
			}
		case "ctts":
			{
				if self.CompositionOffset, err = ReadCompositionOffset(ar); err != nil {
					return
				}
			}
		case "stsc":
			{
				if self.SampleToChunk, err = ReadSampleToChunk(ar); err != nil {
					return
				}
			}
		case "stss":
			{
				if self.SyncSample, err = ReadSyncSample(ar); err != nil {
					return
				}
			}
		case "stco":
			{
				if self.ChunkOffset, err = ReadChunkOffset(ar); err != nil {
					return
				}
			}
		case "stsz":
			{
				if self.SampleSize, err = ReadSampleSize(ar); err != nil {
					return
				}
			}

		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}

type SampleDesc struct {
	Version int
	Flags   int
	Entries []*SampleDescEntry
}

func ReadSampleDesc(r *io.LimitedReader) (res *SampleDesc, err error) {

	self := &SampleDesc{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	var count int
	if count, err = ReadInt(r, 4); err != nil {
		return
	}
	self.Entries = make([]*SampleDescEntry, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadSampleDescEntry(r); err != nil {
			return
		}
	}
	res = self
	return
}

type TimeToSample struct {
	Version int
	Flags   int
	Entries []TimeToSampleEntry
}

func ReadTimeToSample(r *io.LimitedReader) (res *TimeToSample, err error) {

	self := &TimeToSample{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	var count int
	if count, err = ReadInt(r, 4); err != nil {
		return
	}
	self.Entries = make([]TimeToSampleEntry, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadTimeToSampleEntry(r); err != nil {
			return
		}
	}
	res = self
	return
}

type TimeToSampleEntry struct {
	Count    int
	Duration int
}

func ReadTimeToSampleEntry(r *io.LimitedReader) (self TimeToSampleEntry, err error) {

	if self.Count, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadInt(r, 4); err != nil {
		return
	}
	return
}

type SampleToChunk struct {
	Version int
	Flags   int
	Entries []SampleToChunkEntry
}

func ReadSampleToChunk(r *io.LimitedReader) (res *SampleToChunk, err error) {

	self := &SampleToChunk{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	var count int
	if count, err = ReadInt(r, 4); err != nil {
		return
	}
	self.Entries = make([]SampleToChunkEntry, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadSampleToChunkEntry(r); err != nil {
			return
		}
	}
	res = self
	return
}

type SampleToChunkEntry struct {
	FirstChunk      int
	SamplesPerChunk int
	SampleDescId    int
}

func ReadSampleToChunkEntry(r *io.LimitedReader) (self SampleToChunkEntry, err error) {

	if self.FirstChunk, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.SamplesPerChunk, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.SampleDescId, err = ReadInt(r, 4); err != nil {
		return
	}
	return
}

type CompositionOffset struct {
	Version int
	Flags   int
	Entries []int
}

func ReadCompositionOffset(r *io.LimitedReader) (res *CompositionOffset, err error) {

	self := &CompositionOffset{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	var count int
	if count, err = ReadInt(r, 4); err != nil {
		return
	}
	self.Entries = make([]int, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadInt(r, 4); err != nil {
			return
		}
	}
	res = self
	return
}

type CompositionOffsetEntry struct {
	Count  int
	Offset int
}

func ReadCompositionOffsetEntry(r *io.LimitedReader) (self CompositionOffsetEntry, err error) {

	if self.Count, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Offset, err = ReadInt(r, 4); err != nil {
		return
	}
	return
}

type SyncSample struct {
	Version int
	Flags   int
	Entries []int
}

func ReadSyncSample(r *io.LimitedReader) (res *SyncSample, err error) {

	self := &SyncSample{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	var count int
	if count, err = ReadInt(r, 4); err != nil {
		return
	}
	self.Entries = make([]int, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadInt(r, 4); err != nil {
			return
		}
	}
	res = self
	return
}

type SampleSize struct {
	Version int
	Flags   int
	Entries []int
}

func ReadSampleSize(r *io.LimitedReader) (res *SampleSize, err error) {

	self := &SampleSize{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	var count int
	if count, err = ReadInt(r, 4); err != nil {
		return
	}
	self.Entries = make([]int, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadInt(r, 4); err != nil {
			return
		}
	}
	res = self
	return
}

type ChunkOffset struct {
	Version int
	Flags   int
	Entries []int
}

func ReadChunkOffset(r *io.LimitedReader) (res *ChunkOffset, err error) {

	self := &ChunkOffset{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	var count int
	if count, err = ReadInt(r, 4); err != nil {
		return
	}
	self.Entries = make([]int, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadInt(r, 4); err != nil {
			return
		}
	}
	res = self
	return
}
