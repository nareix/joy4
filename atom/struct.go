// THIS FILE IS AUTO GENERATED
package atom

import (
	"io"
)

type FileTypeAtom struct {
	Movie *MovieAtom
}

func ReadFileTypeAtom(r *io.LimitedReader) (res *FileTypeAtom, err error) {
	self := &FileTypeAtom{}
	if self.Movie, err = ReadMovieAtom(r); err != nil {
		return
	}
	res = self
	return
}

type MovieAtom struct {
	Header *MovieHeaderAtom
	Tracks []*TrackAtom
}

func ReadMovieAtom(r *io.LimitedReader) (res *MovieAtom, err error) {
	self := &MovieAtom{}
	if self.Header, err = ReadMovieHeaderAtom(r); err != nil {
		return
	}
	for r.N > 0 {
		var item *TrackAtom
		if item, err = ReadTrackAtom(r); err != nil {
			return
		}
		self.Tracks = append(self.Tracks, item)
	}
	res = self
	return
}

type MovieHeaderAtom struct {
	Version           int
	Flags             int
	CTime             TimeStamp
	MTime             TimeStamp
	TimeScale         int
	Duration          int
	PreferredRate     int
	PreferredVolume   int
	Matrix            []byte
	PreviewTime       TimeStamp
	PreviewDuration   TimeStamp
	PosterTime        TimeStamp
	SelectionTime     TimeStamp
	SelectionDuration TimeStamp
	CurrentTime       TimeStamp
	NextTrackId       int
}

func ReadMovieHeaderAtom(r *io.LimitedReader) (res *MovieHeaderAtom, err error) {
	self := &MovieHeaderAtom{}
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
	if self.TimeScale, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.PreferredRate, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.PreferredVolume, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Matrix, err = ReadBytes(r, 36); err != nil {
		return
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

type TrackAtom struct {
	Header *TrackHeaderAtom
	Media  *MediaAtom
}

func ReadTrackAtom(r *io.LimitedReader) (res *TrackAtom, err error) {
	self := &TrackAtom{}
	if self.Header, err = ReadTrackHeaderAtom(r); err != nil {
		return
	}
	if self.Media, err = ReadMediaAtom(r); err != nil {
		return
	}
	res = self
	return
}

type TrackHeaderAtom struct {
	Version        int
	Flags          int
	CTime          TimeStamp
	MTime          TimeStamp
	TrackId        int
	Duration       int
	Layer          int
	AlternateGroup int
	Volume         int
	Matrix         []byte
	TrackWidth     Fixed32
	TrackHeight    Fixed32
}

func ReadTrackHeaderAtom(r *io.LimitedReader) (res *TrackHeaderAtom, err error) {
	self := &TrackHeaderAtom{}
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
	if self.TrackId, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadInt(r, 4); err != nil {
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
	if self.Matrix, err = ReadBytes(r, 36); err != nil {
		return
	}
	if self.TrackWidth, err = ReadFixed32(r, 4); err != nil {
		return
	}
	if self.TrackHeight, err = ReadFixed32(r, 4); err != nil {
		return
	}
	res = self
	return
}

type MediaAtom struct {
	Header *MediaHeaderAtom
	Info   *MediaInfoAtom
}

func ReadMediaAtom(r *io.LimitedReader) (res *MediaAtom, err error) {
	self := &MediaAtom{}
	if self.Header, err = ReadMediaHeaderAtom(r); err != nil {
		return
	}
	if self.Info, err = ReadMediaInfoAtom(r); err != nil {
		return
	}
	res = self
	return
}

type MediaHeaderAtom struct {
	Version   int
	Flags     int
	CTime     TimeStamp
	MTime     TimeStamp
	TimeScale int
	Duration  int
	Language  int
	Quality   int
}

func ReadMediaHeaderAtom(r *io.LimitedReader) (res *MediaHeaderAtom, err error) {
	self := &MediaHeaderAtom{}
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

type MediaInfoAtom struct {
	Video  *VideoMediaInfoAtom
	Sample *SampleTableAtom
}

func ReadMediaInfoAtom(r *io.LimitedReader) (res *MediaInfoAtom, err error) {
	self := &MediaInfoAtom{}
	if self.Video, err = ReadVideoMediaInfoAtom(r); err != nil {
		return
	}
	if self.Sample, err = ReadSampleTableAtom(r); err != nil {
		return
	}
	res = self
	return
}

type VideoMediaInfoAtom struct {
	Version      int
	Flags        int
	GraphicsMode int
	Opcolor      []int
}

func ReadVideoMediaInfoAtom(r *io.LimitedReader) (res *VideoMediaInfoAtom, err error) {
	self := &VideoMediaInfoAtom{}
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
		var item int
		if item, err = ReadInt(r, 2); err != nil {
			return
		}
		self.Opcolor = append(self.Opcolor, item)
	}
	res = self
	return
}

type SampleTableAtom struct {
	SampleDesc        *SampleDescAtom
	TimeToSample      *TimeToSampleAtom
	CompositionOffset *CompositionOffsetAtom
	SyncSample        *SyncSampleAtom
	SampleSize        *SampleSizeAtom
	ChunkOffset       *ChunkOffsetAtom
}

func ReadSampleTableAtom(r *io.LimitedReader) (res *SampleTableAtom, err error) {
	self := &SampleTableAtom{}
	if self.SampleDesc, err = ReadSampleDescAtom(r); err != nil {
		return
	}
	if self.TimeToSample, err = ReadTimeToSampleAtom(r); err != nil {
		return
	}
	if self.CompositionOffset, err = ReadCompositionOffsetAtom(r); err != nil {
		return
	}
	if self.SyncSample, err = ReadSyncSampleAtom(r); err != nil {
		return
	}
	if self.SampleSize, err = ReadSampleSizeAtom(r); err != nil {
		return
	}
	if self.ChunkOffset, err = ReadChunkOffsetAtom(r); err != nil {
		return
	}
	res = self
	return
}

type SampleDescAtom struct {
	Version int
	Flags   int
	Entries []SampleDescEntry
}

func ReadSampleDescAtom(r *io.LimitedReader) (res *SampleDescAtom, err error) {
	self := &SampleDescAtom{}
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
	for i := 0; i < count; i++ {
		var item SampleDescEntry
		if item, err = ReadSampleDescEntry(r); err != nil {
			return
		}
		self.Entries = append(self.Entries, item)
	}
	res = self
	return
}

type TimeToSampleAtom struct {
	Version int
	Flags   int
	Entries []TimeToSampleEntry
}

func ReadTimeToSampleAtom(r *io.LimitedReader) (res *TimeToSampleAtom, err error) {
	self := &TimeToSampleAtom{}
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
	for i := 0; i < count; i++ {
		var item TimeToSampleEntry
		if item, err = ReadTimeToSampleEntry(r); err != nil {
			return
		}
		self.Entries = append(self.Entries, item)
	}
	res = self
	return
}

type CompositionOffsetAtom struct {
	Version int
	Flags   int
	Entries []CompositionOffsetEntry
}

func ReadCompositionOffsetAtom(r *io.LimitedReader) (res *CompositionOffsetAtom, err error) {
	self := &CompositionOffsetAtom{}
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
	for i := 0; i < count; i++ {
		var item CompositionOffsetEntry
		if item, err = ReadCompositionOffsetEntry(r); err != nil {
			return
		}
		self.Entries = append(self.Entries, item)
	}
	res = self
	return
}

type SyncSampleAtom struct {
	Version int
	Flags   int
	Entries []int
}

func ReadSyncSampleAtom(r *io.LimitedReader) (res *SyncSampleAtom, err error) {
	self := &SyncSampleAtom{}
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
	for i := 0; i < count; i++ {
		var item int
		if item, err = ReadInt(r, 4); err != nil {
			return
		}
		self.Entries = append(self.Entries, item)
	}
	res = self
	return
}

type SampleSizeAtom struct {
	Version int
	Flags   int
	Entries []int
}

func ReadSampleSizeAtom(r *io.LimitedReader) (res *SampleSizeAtom, err error) {
	self := &SampleSizeAtom{}
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
	for i := 0; i < count; i++ {
		var item int
		if item, err = ReadInt(r, 4); err != nil {
			return
		}
		self.Entries = append(self.Entries, item)
	}
	res = self
	return
}

type ChunkOffsetAtom struct {
	Version int
	Flags   int
	Entries []int
}

func ReadChunkOffsetAtom(r *io.LimitedReader) (res *ChunkOffsetAtom, err error) {
	self := &ChunkOffsetAtom{}
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
	for i := 0; i < count; i++ {
		var item int
		if item, err = ReadInt(r, 4); err != nil {
			return
		}
		self.Entries = append(self.Entries, item)
	}
	res = self
	return
}

type SampleDescEntry struct {
	Format     string
	DataRefIdx int
	Data       []byte
}

func ReadSampleDescEntry(r *io.LimitedReader) (self SampleDescEntry, err error) {
	if self.Format, err = ReadString(r, 4); err != nil {
		return
	}
	if self.DataRefIdx, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Data, err = ReadBytesLeft(r); err != nil {
		return
	}
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
