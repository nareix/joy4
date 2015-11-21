// THIS FILE IS AUTO GENERATED
package atom

import (
	"io"
	"log"
)

type FileType struct {
}

func ReadFileType(r *io.LimitedReader) (res *FileType, err error) {
	log.Println("ReadFileType")
	self := &FileType{}

	res = self
	return
}
func WriteFileType(w io.WriteSeeker, self *FileType) (err error) {
	log.Println("WriteFileType")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "ftyp"); err != nil {
		return
	}
	w = aw

	if err = aw.Close(); err != nil {
		return
	}
	return
}

type Movie struct {
	Header *MovieHeader
	Tracks []*Track
}

func ReadMovie(r *io.LimitedReader) (res *Movie, err error) {
	log.Println("ReadMovie")
	self := &Movie{}
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
		default:
			{
				log.Println("skip", cc4)
			}
		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}
func WriteMovie(w io.WriteSeeker, self *Movie) (err error) {
	log.Println("WriteMovie")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "moov"); err != nil {
		return
	}
	w = aw
	if self.Header != nil {
		if err = WriteMovieHeader(w, self.Header); err != nil {
			return
		}
	}
	if self.Tracks != nil {
		for _, elem := range self.Tracks {
			if err = WriteTrack(w, elem); err != nil {
				return
			}
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
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
	log.Println("ReadMovieHeader")
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
func WriteMovieHeader(w io.WriteSeeker, self *MovieHeader) (err error) {
	log.Println("WriteMovieHeader")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "mvhd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.CTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.MTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.TimeScale, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.Duration, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.PreferredRate, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.PreferredVolume, 2); err != nil {
		return
	}
	if err = WriteDummy(w, 10); err != nil {
		return
	}
	for _, elem := range self.Matrix {
		if err = WriteInt(w, elem, 4); err != nil {
			return
		}
	}
	if err = WriteTimeStamp(w, self.PreviewTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.PreviewDuration, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.PosterTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.SelectionTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.SelectionDuration, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.CurrentTime, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.NextTrackId, 4); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type Track struct {
	Header *TrackHeader
	Media  *Media
}

func ReadTrack(r *io.LimitedReader) (res *Track, err error) {
	log.Println("ReadTrack")
	self := &Track{}
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
		default:
			{
				log.Println("skip", cc4)
			}
		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}
func WriteTrack(w io.WriteSeeker, self *Track) (err error) {
	log.Println("WriteTrack")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "trak"); err != nil {
		return
	}
	w = aw
	if self.Header != nil {
		if err = WriteTrackHeader(w, self.Header); err != nil {
			return
		}
	}
	if self.Media != nil {
		if err = WriteMedia(w, self.Media); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
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
	log.Println("ReadTrackHeader")
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
func WriteTrackHeader(w io.WriteSeeker, self *TrackHeader) (err error) {
	log.Println("WriteTrackHeader")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "tkhd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.CTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.MTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.TrackId, 4); err != nil {
		return
	}
	if err = WriteDummy(w, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.Duration, 4); err != nil {
		return
	}
	if err = WriteDummy(w, 8); err != nil {
		return
	}
	if err = WriteInt(w, self.Layer, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.AlternateGroup, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Volume, 2); err != nil {
		return
	}
	if err = WriteDummy(w, 2); err != nil {
		return
	}
	for _, elem := range self.Matrix {
		if err = WriteInt(w, elem, 4); err != nil {
			return
		}
	}
	if err = WriteInt(w, self.TrackWidth, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.TrackHeader, 4); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type Media struct {
	Header *MediaHeader
	Info   *MediaInfo
	Hdlr   *HandlerRefer
}

func ReadMedia(r *io.LimitedReader) (res *Media, err error) {
	log.Println("ReadMedia")
	self := &Media{}
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
		case "hdlr":
			{
				if self.Hdlr, err = ReadHandlerRefer(ar); err != nil {
					return
				}
			}
		default:
			{
				log.Println("skip", cc4)
			}
		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}
func WriteMedia(w io.WriteSeeker, self *Media) (err error) {
	log.Println("WriteMedia")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "mdia"); err != nil {
		return
	}
	w = aw
	if self.Header != nil {
		if err = WriteMediaHeader(w, self.Header); err != nil {
			return
		}
	}
	if self.Info != nil {
		if err = WriteMediaInfo(w, self.Info); err != nil {
			return
		}
	}
	if self.Hdlr != nil {
		if err = WriteHandlerRefer(w, self.Hdlr); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
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
	log.Println("ReadMediaHeader")
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
func WriteMediaHeader(w io.WriteSeeker, self *MediaHeader) (err error) {
	log.Println("WriteMediaHeader")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "mdhd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, self.CTime, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.MTime, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.TimeScale, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Duration, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Language, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Quality, 2); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type MediaInfo struct {
	Sound  *SoundMediaInfo
	Video  *VideoMediaInfo
	Sample *SampleTable
}

func ReadMediaInfo(r *io.LimitedReader) (res *MediaInfo, err error) {
	log.Println("ReadMediaInfo")
	self := &MediaInfo{}
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
		default:
			{
				log.Println("skip", cc4)
			}
		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}
func WriteMediaInfo(w io.WriteSeeker, self *MediaInfo) (err error) {
	log.Println("WriteMediaInfo")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "minf"); err != nil {
		return
	}
	w = aw
	if self.Sound != nil {
		if err = WriteSoundMediaInfo(w, self.Sound); err != nil {
			return
		}
	}
	if self.Video != nil {
		if err = WriteVideoMediaInfo(w, self.Video); err != nil {
			return
		}
	}
	if self.Sample != nil {
		if err = WriteSampleTable(w, self.Sample); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type SoundMediaInfo struct {
	Version int
	Flags   int
	Balance int
}

func ReadSoundMediaInfo(r *io.LimitedReader) (res *SoundMediaInfo, err error) {
	log.Println("ReadSoundMediaInfo")
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
func WriteSoundMediaInfo(w io.WriteSeeker, self *SoundMediaInfo) (err error) {
	log.Println("WriteSoundMediaInfo")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "smhd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, self.Balance, 2); err != nil {
		return
	}
	if err = WriteDummy(w, 2); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type VideoMediaInfo struct {
	Version      int
	Flags        int
	GraphicsMode int
	Opcolor      [3]int
}

func ReadVideoMediaInfo(r *io.LimitedReader) (res *VideoMediaInfo, err error) {
	log.Println("ReadVideoMediaInfo")
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
func WriteVideoMediaInfo(w io.WriteSeeker, self *VideoMediaInfo) (err error) {
	log.Println("WriteVideoMediaInfo")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "vmhd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, self.GraphicsMode, 2); err != nil {
		return
	}
	for _, elem := range self.Opcolor {
		if err = WriteInt(w, elem, 2); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
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
	log.Println("ReadSampleTable")
	self := &SampleTable{}
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
		default:
			{
				log.Println("skip", cc4)
			}
		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}
func WriteSampleTable(w io.WriteSeeker, self *SampleTable) (err error) {
	log.Println("WriteSampleTable")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stbl"); err != nil {
		return
	}
	w = aw
	if self.SampleDesc != nil {
		if err = WriteSampleDesc(w, self.SampleDesc); err != nil {
			return
		}
	}
	if self.TimeToSample != nil {
		if err = WriteTimeToSample(w, self.TimeToSample); err != nil {
			return
		}
	}
	if self.CompositionOffset != nil {
		if err = WriteCompositionOffset(w, self.CompositionOffset); err != nil {
			return
		}
	}
	if self.SampleToChunk != nil {
		if err = WriteSampleToChunk(w, self.SampleToChunk); err != nil {
			return
		}
	}
	if self.SyncSample != nil {
		if err = WriteSyncSample(w, self.SyncSample); err != nil {
			return
		}
	}
	if self.ChunkOffset != nil {
		if err = WriteChunkOffset(w, self.ChunkOffset); err != nil {
			return
		}
	}
	if self.SampleSize != nil {
		if err = WriteSampleSize(w, self.SampleSize); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type SampleDesc struct {
	Version int
	Flags   int
	Entries []*SampleDescEntry
}

func ReadSampleDesc(r *io.LimitedReader) (res *SampleDesc, err error) {
	log.Println("ReadSampleDesc")
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
func WriteSampleDesc(w io.WriteSeeker, self *SampleDesc) (err error) {
	log.Println("WriteSampleDesc")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stsd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, len(self.Entries), 4); err != nil {
		return
	}
	for _, elem := range self.Entries {
		if err = WriteSampleDescEntry(w, elem); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type TimeToSample struct {
	Version int
	Flags   int
	Entries []TimeToSampleEntry
}

func ReadTimeToSample(r *io.LimitedReader) (res *TimeToSample, err error) {
	log.Println("ReadTimeToSample")
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
func WriteTimeToSample(w io.WriteSeeker, self *TimeToSample) (err error) {
	log.Println("WriteTimeToSample")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stts"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, len(self.Entries), 4); err != nil {
		return
	}
	for _, elem := range self.Entries {
		if err = WriteTimeToSampleEntry(w, elem); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type TimeToSampleEntry struct {
	Count    int
	Duration int
}

func ReadTimeToSampleEntry(r *io.LimitedReader) (self TimeToSampleEntry, err error) {
	log.Println("ReadTimeToSampleEntry")
	if self.Count, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadInt(r, 4); err != nil {
		return
	}
	return
}
func WriteTimeToSampleEntry(w io.WriteSeeker, self TimeToSampleEntry) (err error) {
	log.Println("WriteTimeToSampleEntry")
	if err = WriteInt(w, self.Count, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Duration, 4); err != nil {
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
	log.Println("ReadSampleToChunk")
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
func WriteSampleToChunk(w io.WriteSeeker, self *SampleToChunk) (err error) {
	log.Println("WriteSampleToChunk")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stsc"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, len(self.Entries), 4); err != nil {
		return
	}
	for _, elem := range self.Entries {
		if err = WriteSampleToChunkEntry(w, elem); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type SampleToChunkEntry struct {
	FirstChunk      int
	SamplesPerChunk int
	SampleDescId    int
}

func ReadSampleToChunkEntry(r *io.LimitedReader) (self SampleToChunkEntry, err error) {
	log.Println("ReadSampleToChunkEntry")
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
func WriteSampleToChunkEntry(w io.WriteSeeker, self SampleToChunkEntry) (err error) {
	log.Println("WriteSampleToChunkEntry")
	if err = WriteInt(w, self.FirstChunk, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.SamplesPerChunk, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.SampleDescId, 4); err != nil {
		return
	}
	return
}

type CompositionOffset struct {
	Version int
	Flags   int
	Entries []CompositionOffsetEntry
}

func ReadCompositionOffset(r *io.LimitedReader) (res *CompositionOffset, err error) {
	log.Println("ReadCompositionOffset")
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
	self.Entries = make([]CompositionOffsetEntry, count)
	for i := 0; i < count; i++ {
		if self.Entries[i], err = ReadCompositionOffsetEntry(r); err != nil {
			return
		}
	}
	res = self
	return
}
func WriteCompositionOffset(w io.WriteSeeker, self *CompositionOffset) (err error) {
	log.Println("WriteCompositionOffset")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "ctts"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, len(self.Entries), 4); err != nil {
		return
	}
	for _, elem := range self.Entries {
		if err = WriteCompositionOffsetEntry(w, elem); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type CompositionOffsetEntry struct {
	Count  int
	Offset int
}

func ReadCompositionOffsetEntry(r *io.LimitedReader) (self CompositionOffsetEntry, err error) {
	log.Println("ReadCompositionOffsetEntry")
	if self.Count, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Offset, err = ReadInt(r, 4); err != nil {
		return
	}
	return
}
func WriteCompositionOffsetEntry(w io.WriteSeeker, self CompositionOffsetEntry) (err error) {
	log.Println("WriteCompositionOffsetEntry")
	if err = WriteInt(w, self.Count, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Offset, 4); err != nil {
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
	log.Println("ReadSyncSample")
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
func WriteSyncSample(w io.WriteSeeker, self *SyncSample) (err error) {
	log.Println("WriteSyncSample")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stss"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, len(self.Entries), 4); err != nil {
		return
	}
	for _, elem := range self.Entries {
		if err = WriteInt(w, elem, 4); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type SampleSize struct {
	Version    int
	Flags      int
	SampleSize int
	Entries    []int
}

func ReadSampleSize(r *io.LimitedReader) (res *SampleSize, err error) {
	log.Println("ReadSampleSize")
	self := &SampleSize{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.SampleSize, err = ReadInt(r, 4); err != nil {
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
func WriteSampleSize(w io.WriteSeeker, self *SampleSize) (err error) {
	log.Println("WriteSampleSize")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stsz"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, self.SampleSize, 4); err != nil {
		return
	}
	if err = WriteInt(w, len(self.Entries), 4); err != nil {
		return
	}
	for _, elem := range self.Entries {
		if err = WriteInt(w, elem, 4); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type ChunkOffset struct {
	Version int
	Flags   int
	Entries []int
}

func ReadChunkOffset(r *io.LimitedReader) (res *ChunkOffset, err error) {
	log.Println("ReadChunkOffset")
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
func WriteChunkOffset(w io.WriteSeeker, self *ChunkOffset) (err error) {
	log.Println("WriteChunkOffset")
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stco"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, len(self.Entries), 4); err != nil {
		return
	}
	for _, elem := range self.Entries {
		if err = WriteInt(w, elem, 4); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
