package atom

type FileTypeAtom struct {
	Movie *Movie
}

type MovieAtom struct {
	MovieHeader *MovieHeader
}

type MovieHeaderAtom struct {
	Version         int
	Flags           int
	CTime           Ts
	MTime           Ts
	TimeScale       int
	Duration        int
	PreferredRate   int
	PreferredVolume int

	Matrix            []byte
	PreviewTime       Ts
	PreviewDuration   Ts
	PosterTime        Ts
	SelectionTime     Ts
	SelectionDuration Ts
	CurrentTime       Ts
	NextTrackId       int
}

type TrackAtom struct {
	TrackHeader *TrackHeader
	Media       *Media
}

type TrackHeaderAtom struct {
	Version int
	Flags   int
	CTime   Ts
	MTime   Ts
	TrackId int

	Duration int

	Layer          int
	AlternateGroup int
	Volume         int

	Matrix      []byte
	TrackWidth  fixed32
	TrackHeight fixed32
}

type MediaAtom struct {
	MediaHeader *MediaHeader
	MediaInfo   *MediaInfo
}

type MediaHeaderAtom struct {
	Version   int
	Flags     int
	CTime     Ts
	MTime     Ts
	TimeScale int
	Duration  int
	Language  int
	Quality   int
}

type MediaInfoAtom struct {
	VideoMediaInfo *VideoMediaInfo
}

type VideoMediaInfoAtom struct {
	Version      int
	Flags        int
	GraphicsMode int
}

type SampleTableAtom struct {
	SampleDesc        *SampleDesc
	TimeToSample      *TimeToSample
	CompositionOffset *CompositionOffset
	SyncSample        *SyncSample
	SampleSize        *SampleSize
	ChunkOffset       *ChunkOffset
}

type SampleDescAtom struct {
	Version int
	Flags   int
}

type TimeToSampleAtom struct {
	Version int
	Flags   int
}

type CompositionOffsetAtom struct {
	Version int
	Flags   int
}

type SyncSampleAtom struct {
	Version int
	Flags   int
}

type SampleSizeAtom struct {
	Version int
	Flags   int
}

type ChunkOffsetAtom struct {
	Version int
	Flags   int
}

type sampleDescEntry struct {
	Format string

	DataRefIdx int
	Data       []byte
}

type timeToSampleEntry struct {
	Count    int
	Duration int
}

type compositionOffsetEntry struct {
	Count  int
	Offset int
}
