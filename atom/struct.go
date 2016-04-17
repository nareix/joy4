// THIS FILE IS AUTO GENERATED
package atom

import (
	"io"
)

type Movie struct {
	Header *MovieHeader
	Iods   *Iods
	Tracks []*Track
}

func ReadMovie(r *io.LimitedReader) (res *Movie, err error) {

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
		case "iods":
			{
				if self.Iods, err = ReadIods(ar); err != nil {
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
func WriteMovie(w io.WriteSeeker, self *Movie) (err error) {

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
	if self.Iods != nil {
		if err = WriteIods(w, self.Iods); err != nil {
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
func WalkMovie(w Walker, self *Movie) {

	w.StartStruct("Movie")
	if self.Header != nil {
		WalkMovieHeader(w, self.Header)
	}
	if self.Iods != nil {
		WalkIods(w, self.Iods)
	}
	for i, item := range self.Tracks {
		if w.FilterArrayItem("Movie", "Tracks", i, len(self.Tracks)) {
			if item != nil {
				WalkTrack(w, item)
			}
		} else {
			w.ArrayLeft(i, len(self.Tracks))
			break
		}
	}
	w.EndStruct()
	return
}

type Iods struct {
	Data []byte
}

func ReadIods(r *io.LimitedReader) (res *Iods, err error) {

	self := &Iods{}
	if self.Data, err = ReadBytes(r, int(r.N)); err != nil {
		return
	}
	res = self
	return
}
func WriteIods(w io.WriteSeeker, self *Iods) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "iods"); err != nil {
		return
	}
	w = aw
	if err = WriteBytes(w, self.Data, len(self.Data)); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkIods(w Walker, self *Iods) {

	w.StartStruct("Iods")
	w.Name("Data")
	w.Bytes(self.Data)
	w.EndStruct()
	return
}

type MovieHeader struct {
	Version           int
	Flags             int
	CreateTime        TimeStamp
	ModifyTime        TimeStamp
	TimeScale         int
	Duration          int
	PreferredRate     Fixed
	PreferredVolume   Fixed
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
	if self.CreateTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.ModifyTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.TimeScale, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.PreferredRate, err = ReadFixed(r, 4); err != nil {
		return
	}
	if self.PreferredVolume, err = ReadFixed(r, 2); err != nil {
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
	if err = WriteTimeStamp(w, self.CreateTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.ModifyTime, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.TimeScale, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Duration, 4); err != nil {
		return
	}
	if err = WriteFixed(w, self.PreferredRate, 4); err != nil {
		return
	}
	if err = WriteFixed(w, self.PreferredVolume, 2); err != nil {
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
func WalkMovieHeader(w Walker, self *MovieHeader) {

	w.StartStruct("MovieHeader")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("CreateTime")
	w.TimeStamp(self.CreateTime)
	w.Name("ModifyTime")
	w.TimeStamp(self.ModifyTime)
	w.Name("TimeScale")
	w.Int(self.TimeScale)
	w.Name("Duration")
	w.Int(self.Duration)
	w.Name("PreferredRate")
	w.Fixed(self.PreferredRate)
	w.Name("PreferredVolume")
	w.Fixed(self.PreferredVolume)
	for i, item := range self.Matrix {
		if w.FilterArrayItem("MovieHeader", "Matrix", i, len(self.Matrix)) {
			w.Name("Matrix")
			w.Int(item)
		} else {
			w.ArrayLeft(i, len(self.Matrix))
			break
		}
	}
	w.Name("PreviewTime")
	w.TimeStamp(self.PreviewTime)
	w.Name("PreviewDuration")
	w.TimeStamp(self.PreviewDuration)
	w.Name("PosterTime")
	w.TimeStamp(self.PosterTime)
	w.Name("SelectionTime")
	w.TimeStamp(self.SelectionTime)
	w.Name("SelectionDuration")
	w.TimeStamp(self.SelectionDuration)
	w.Name("CurrentTime")
	w.TimeStamp(self.CurrentTime)
	w.Name("NextTrackId")
	w.Int(self.NextTrackId)
	w.EndStruct()
	return
}

type Track struct {
	Header *TrackHeader
	Media  *Media
}

func ReadTrack(r *io.LimitedReader) (res *Track, err error) {

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

		}
		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}
	res = self
	return
}
func WriteTrack(w io.WriteSeeker, self *Track) (err error) {

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
func WalkTrack(w Walker, self *Track) {

	w.StartStruct("Track")
	if self.Header != nil {
		WalkTrackHeader(w, self.Header)
	}
	if self.Media != nil {
		WalkMedia(w, self.Media)
	}
	w.EndStruct()
	return
}

type TrackHeader struct {
	Version        int
	Flags          int
	CreateTime     TimeStamp
	ModifyTime     TimeStamp
	TrackId        int
	Duration       int
	Layer          int
	AlternateGroup int
	Volume         Fixed
	Matrix         [9]int
	TrackWidth     Fixed
	TrackHeight    Fixed
}

func ReadTrackHeader(r *io.LimitedReader) (res *TrackHeader, err error) {

	self := &TrackHeader{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.CreateTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.ModifyTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.TrackId, err = ReadInt(r, 4); err != nil {
		return
	}
	if _, err = ReadDummy(r, 4); err != nil {
		return
	}
	if self.Duration, err = ReadInt(r, 4); err != nil {
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
	if self.Volume, err = ReadFixed(r, 2); err != nil {
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
	if self.TrackWidth, err = ReadFixed(r, 4); err != nil {
		return
	}
	if self.TrackHeight, err = ReadFixed(r, 4); err != nil {
		return
	}
	res = self
	return
}
func WriteTrackHeader(w io.WriteSeeker, self *TrackHeader) (err error) {

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
	if err = WriteTimeStamp(w, self.CreateTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.ModifyTime, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.TrackId, 4); err != nil {
		return
	}
	if err = WriteDummy(w, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Duration, 4); err != nil {
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
	if err = WriteFixed(w, self.Volume, 2); err != nil {
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
	if err = WriteFixed(w, self.TrackWidth, 4); err != nil {
		return
	}
	if err = WriteFixed(w, self.TrackHeight, 4); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkTrackHeader(w Walker, self *TrackHeader) {

	w.StartStruct("TrackHeader")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("CreateTime")
	w.TimeStamp(self.CreateTime)
	w.Name("ModifyTime")
	w.TimeStamp(self.ModifyTime)
	w.Name("TrackId")
	w.Int(self.TrackId)
	w.Name("Duration")
	w.Int(self.Duration)
	w.Name("Layer")
	w.Int(self.Layer)
	w.Name("AlternateGroup")
	w.Int(self.AlternateGroup)
	w.Name("Volume")
	w.Fixed(self.Volume)
	for i, item := range self.Matrix {
		if w.FilterArrayItem("TrackHeader", "Matrix", i, len(self.Matrix)) {
			w.Name("Matrix")
			w.Int(item)
		} else {
			w.ArrayLeft(i, len(self.Matrix))
			break
		}
	}
	w.Name("TrackWidth")
	w.Fixed(self.TrackWidth)
	w.Name("TrackHeight")
	w.Fixed(self.TrackHeight)
	w.EndStruct()
	return
}

type HandlerRefer struct {
	Version int
	Flags   int
	Type    string
	SubType string
	Name    string
}

func ReadHandlerRefer(r *io.LimitedReader) (res *HandlerRefer, err error) {

	self := &HandlerRefer{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.Type, err = ReadString(r, 4); err != nil {
		return
	}
	if self.SubType, err = ReadString(r, 4); err != nil {
		return
	}
	if self.Name, err = ReadString(r, int(r.N)); err != nil {
		return
	}
	res = self
	return
}
func WriteHandlerRefer(w io.WriteSeeker, self *HandlerRefer) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "hdlr"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteString(w, self.Type, 4); err != nil {
		return
	}
	if err = WriteString(w, self.SubType, 4); err != nil {
		return
	}
	if err = WriteString(w, self.Name, len(self.Name)); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkHandlerRefer(w Walker, self *HandlerRefer) {

	w.StartStruct("HandlerRefer")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("Type")
	w.String(self.Type)
	w.Name("SubType")
	w.String(self.SubType)
	w.Name("Name")
	w.String(self.Name)
	w.EndStruct()
	return
}

type Media struct {
	Header  *MediaHeader
	Handler *HandlerRefer
	Info    *MediaInfo
}

func ReadMedia(r *io.LimitedReader) (res *Media, err error) {

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
		case "hdlr":
			{
				if self.Handler, err = ReadHandlerRefer(ar); err != nil {
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
func WriteMedia(w io.WriteSeeker, self *Media) (err error) {

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
	if self.Handler != nil {
		if err = WriteHandlerRefer(w, self.Handler); err != nil {
			return
		}
	}
	if self.Info != nil {
		if err = WriteMediaInfo(w, self.Info); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkMedia(w Walker, self *Media) {

	w.StartStruct("Media")
	if self.Header != nil {
		WalkMediaHeader(w, self.Header)
	}
	if self.Handler != nil {
		WalkHandlerRefer(w, self.Handler)
	}
	if self.Info != nil {
		WalkMediaInfo(w, self.Info)
	}
	w.EndStruct()
	return
}

type MediaHeader struct {
	Version    int
	Flags      int
	CreateTime TimeStamp
	ModifyTime TimeStamp
	TimeScale  int
	Duration   int
	Language   int
	Quality    int
}

func ReadMediaHeader(r *io.LimitedReader) (res *MediaHeader, err error) {

	self := &MediaHeader{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.CreateTime, err = ReadTimeStamp(r, 4); err != nil {
		return
	}
	if self.ModifyTime, err = ReadTimeStamp(r, 4); err != nil {
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
	if err = WriteTimeStamp(w, self.CreateTime, 4); err != nil {
		return
	}
	if err = WriteTimeStamp(w, self.ModifyTime, 4); err != nil {
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
func WalkMediaHeader(w Walker, self *MediaHeader) {

	w.StartStruct("MediaHeader")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("CreateTime")
	w.TimeStamp(self.CreateTime)
	w.Name("ModifyTime")
	w.TimeStamp(self.ModifyTime)
	w.Name("TimeScale")
	w.Int(self.TimeScale)
	w.Name("Duration")
	w.Int(self.Duration)
	w.Name("Language")
	w.Int(self.Language)
	w.Name("Quality")
	w.Int(self.Quality)
	w.EndStruct()
	return
}

type MediaInfo struct {
	Sound  *SoundMediaInfo
	Video  *VideoMediaInfo
	Data   *DataInfo
	Sample *SampleTable
}

func ReadMediaInfo(r *io.LimitedReader) (res *MediaInfo, err error) {

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
		case "dinf":
			{
				if self.Data, err = ReadDataInfo(ar); err != nil {
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
func WriteMediaInfo(w io.WriteSeeker, self *MediaInfo) (err error) {

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
	if self.Data != nil {
		if err = WriteDataInfo(w, self.Data); err != nil {
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
func WalkMediaInfo(w Walker, self *MediaInfo) {

	w.StartStruct("MediaInfo")
	if self.Sound != nil {
		WalkSoundMediaInfo(w, self.Sound)
	}
	if self.Video != nil {
		WalkVideoMediaInfo(w, self.Video)
	}
	if self.Data != nil {
		WalkDataInfo(w, self.Data)
	}
	if self.Sample != nil {
		WalkSampleTable(w, self.Sample)
	}
	w.EndStruct()
	return
}

type DataInfo struct {
	Refer *DataRefer
}

func ReadDataInfo(r *io.LimitedReader) (res *DataInfo, err error) {

	self := &DataInfo{}
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "dref":
			{
				if self.Refer, err = ReadDataRefer(ar); err != nil {
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
func WriteDataInfo(w io.WriteSeeker, self *DataInfo) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "dinf"); err != nil {
		return
	}
	w = aw
	if self.Refer != nil {
		if err = WriteDataRefer(w, self.Refer); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkDataInfo(w Walker, self *DataInfo) {

	w.StartStruct("DataInfo")
	if self.Refer != nil {
		WalkDataRefer(w, self.Refer)
	}
	w.EndStruct()
	return
}

type DataRefer struct {
	Version int
	Flags   int
	Url     *DataReferUrl
}

func ReadDataRefer(r *io.LimitedReader) (res *DataRefer, err error) {

	self := &DataRefer{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if _, err = ReadDummy(r, 4); err != nil {
		return
	}
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "url ":
			{
				if self.Url, err = ReadDataReferUrl(ar); err != nil {
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
func WriteDataRefer(w io.WriteSeeker, self *DataRefer) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "dref"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	var atomsCount int
	var atomsCountPos int64
	if atomsCountPos, err = WriteEmptyInt(w, 4); err != nil {
		return
	}
	if self.Url != nil {
		if err = WriteDataReferUrl(w, self.Url); err != nil {
			return
		}
		atomsCount++
	}
	if err = RefillInt(w, atomsCountPos, atomsCount, 4); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkDataRefer(w Walker, self *DataRefer) {

	w.StartStruct("DataRefer")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	if self.Url != nil {
		WalkDataReferUrl(w, self.Url)
	}
	w.EndStruct()
	return
}

type DataReferUrl struct {
	Version int
	Flags   int
}

func ReadDataReferUrl(r *io.LimitedReader) (res *DataReferUrl, err error) {

	self := &DataReferUrl{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	res = self
	return
}
func WriteDataReferUrl(w io.WriteSeeker, self *DataReferUrl) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "url "); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkDataReferUrl(w Walker, self *DataReferUrl) {

	w.StartStruct("DataReferUrl")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.EndStruct()
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
func WriteSoundMediaInfo(w io.WriteSeeker, self *SoundMediaInfo) (err error) {

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
func WalkSoundMediaInfo(w Walker, self *SoundMediaInfo) {

	w.StartStruct("SoundMediaInfo")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("Balance")
	w.Int(self.Balance)
	w.EndStruct()
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
func WriteVideoMediaInfo(w io.WriteSeeker, self *VideoMediaInfo) (err error) {

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
func WalkVideoMediaInfo(w Walker, self *VideoMediaInfo) {

	w.StartStruct("VideoMediaInfo")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("GraphicsMode")
	w.Int(self.GraphicsMode)
	for i, item := range self.Opcolor {
		if w.FilterArrayItem("VideoMediaInfo", "Opcolor", i, len(self.Opcolor)) {
			w.Name("Opcolor")
			w.Int(item)
		} else {
			w.ArrayLeft(i, len(self.Opcolor))
			break
		}
	}
	w.EndStruct()
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
func WriteSampleTable(w io.WriteSeeker, self *SampleTable) (err error) {

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
func WalkSampleTable(w Walker, self *SampleTable) {

	w.StartStruct("SampleTable")
	if self.SampleDesc != nil {
		WalkSampleDesc(w, self.SampleDesc)
	}
	if self.TimeToSample != nil {
		WalkTimeToSample(w, self.TimeToSample)
	}
	if self.CompositionOffset != nil {
		WalkCompositionOffset(w, self.CompositionOffset)
	}
	if self.SampleToChunk != nil {
		WalkSampleToChunk(w, self.SampleToChunk)
	}
	if self.SyncSample != nil {
		WalkSyncSample(w, self.SyncSample)
	}
	if self.ChunkOffset != nil {
		WalkChunkOffset(w, self.ChunkOffset)
	}
	if self.SampleSize != nil {
		WalkSampleSize(w, self.SampleSize)
	}
	w.EndStruct()
	return
}

type SampleDesc struct {
	Version  int
	Avc1Desc *Avc1Desc
	Mp4aDesc *Mp4aDesc
}

func ReadSampleDesc(r *io.LimitedReader) (res *SampleDesc, err error) {

	self := &SampleDesc{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if _, err = ReadDummy(r, 3); err != nil {
		return
	}
	if _, err = ReadDummy(r, 4); err != nil {
		return
	}
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "avc1":
			{
				if self.Avc1Desc, err = ReadAvc1Desc(ar); err != nil {
					return
				}
			}
		case "mp4a":
			{
				if self.Mp4aDesc, err = ReadMp4aDesc(ar); err != nil {
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
func WriteSampleDesc(w io.WriteSeeker, self *SampleDesc) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "stsd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteDummy(w, 3); err != nil {
		return
	}
	var atomsCount int
	var atomsCountPos int64
	if atomsCountPos, err = WriteEmptyInt(w, 4); err != nil {
		return
	}
	if self.Avc1Desc != nil {
		if err = WriteAvc1Desc(w, self.Avc1Desc); err != nil {
			return
		}
		atomsCount++
	}
	if self.Mp4aDesc != nil {
		if err = WriteMp4aDesc(w, self.Mp4aDesc); err != nil {
			return
		}
		atomsCount++
	}
	if err = RefillInt(w, atomsCountPos, atomsCount, 4); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkSampleDesc(w Walker, self *SampleDesc) {

	w.StartStruct("SampleDesc")
	w.Name("Version")
	w.Int(self.Version)
	if self.Avc1Desc != nil {
		WalkAvc1Desc(w, self.Avc1Desc)
	}
	if self.Mp4aDesc != nil {
		WalkMp4aDesc(w, self.Mp4aDesc)
	}
	w.EndStruct()
	return
}

type Mp4aDesc struct {
	DataRefIdx       int
	Version          int
	RevisionLevel    int
	Vendor           int
	NumberOfChannels int
	SampleSize       int
	CompressionId    int
	SampleRate       Fixed
	Conf             *ElemStreamDesc
}

func ReadMp4aDesc(r *io.LimitedReader) (res *Mp4aDesc, err error) {

	self := &Mp4aDesc{}
	if _, err = ReadDummy(r, 6); err != nil {
		return
	}
	if self.DataRefIdx, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Version, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.RevisionLevel, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Vendor, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.NumberOfChannels, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.SampleSize, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.CompressionId, err = ReadInt(r, 2); err != nil {
		return
	}
	if _, err = ReadDummy(r, 2); err != nil {
		return
	}
	if self.SampleRate, err = ReadFixed(r, 4); err != nil {
		return
	}
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "esds":
			{
				if self.Conf, err = ReadElemStreamDesc(ar); err != nil {
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
func WriteMp4aDesc(w io.WriteSeeker, self *Mp4aDesc) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "mp4a"); err != nil {
		return
	}
	w = aw
	if err = WriteDummy(w, 6); err != nil {
		return
	}
	if err = WriteInt(w, self.DataRefIdx, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Version, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.RevisionLevel, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Vendor, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.NumberOfChannels, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.SampleSize, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.CompressionId, 2); err != nil {
		return
	}
	if err = WriteDummy(w, 2); err != nil {
		return
	}
	if err = WriteFixed(w, self.SampleRate, 4); err != nil {
		return
	}
	if self.Conf != nil {
		if err = WriteElemStreamDesc(w, self.Conf); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkMp4aDesc(w Walker, self *Mp4aDesc) {

	w.StartStruct("Mp4aDesc")
	w.Name("DataRefIdx")
	w.Int(self.DataRefIdx)
	w.Name("Version")
	w.Int(self.Version)
	w.Name("RevisionLevel")
	w.Int(self.RevisionLevel)
	w.Name("Vendor")
	w.Int(self.Vendor)
	w.Name("NumberOfChannels")
	w.Int(self.NumberOfChannels)
	w.Name("SampleSize")
	w.Int(self.SampleSize)
	w.Name("CompressionId")
	w.Int(self.CompressionId)
	w.Name("SampleRate")
	w.Fixed(self.SampleRate)
	if self.Conf != nil {
		WalkElemStreamDesc(w, self.Conf)
	}
	w.EndStruct()
	return
}

type ElemStreamDesc struct {
	Version int
	Data    []byte
}

func ReadElemStreamDesc(r *io.LimitedReader) (res *ElemStreamDesc, err error) {

	self := &ElemStreamDesc{}
	if self.Version, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Data, err = ReadBytes(r, int(r.N)); err != nil {
		return
	}
	res = self
	return
}
func WriteElemStreamDesc(w io.WriteSeeker, self *ElemStreamDesc) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "esds"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 4); err != nil {
		return
	}
	if err = WriteBytes(w, self.Data, len(self.Data)); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkElemStreamDesc(w Walker, self *ElemStreamDesc) {

	w.StartStruct("ElemStreamDesc")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Data")
	w.Bytes(self.Data)
	w.EndStruct()
	return
}

type Avc1Desc struct {
	DataRefIdx           int
	Version              int
	Revision             int
	Vendor               int
	TemporalQuality      int
	SpatialQuality       int
	Width                int
	Height               int
	HorizontalResolution Fixed
	VorizontalResolution Fixed
	FrameCount           int
	CompressorName       string
	Depth                int
	ColorTableId         int
	Conf                 *Avc1Conf
}

func ReadAvc1Desc(r *io.LimitedReader) (res *Avc1Desc, err error) {

	self := &Avc1Desc{}
	if _, err = ReadDummy(r, 6); err != nil {
		return
	}
	if self.DataRefIdx, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Version, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Revision, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Vendor, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.TemporalQuality, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.SpatialQuality, err = ReadInt(r, 4); err != nil {
		return
	}
	if self.Width, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Height, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.HorizontalResolution, err = ReadFixed(r, 4); err != nil {
		return
	}
	if self.VorizontalResolution, err = ReadFixed(r, 4); err != nil {
		return
	}
	if _, err = ReadDummy(r, 4); err != nil {
		return
	}
	if self.FrameCount, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.CompressorName, err = ReadString(r, 32); err != nil {
		return
	}
	if self.Depth, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.ColorTableId, err = ReadInt(r, 2); err != nil {
		return
	}
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "avcC":
			{
				if self.Conf, err = ReadAvc1Conf(ar); err != nil {
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
func WriteAvc1Desc(w io.WriteSeeker, self *Avc1Desc) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "avc1"); err != nil {
		return
	}
	w = aw
	if err = WriteDummy(w, 6); err != nil {
		return
	}
	if err = WriteInt(w, self.DataRefIdx, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Version, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Revision, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Vendor, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.TemporalQuality, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.SpatialQuality, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Width, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.Height, 2); err != nil {
		return
	}
	if err = WriteFixed(w, self.HorizontalResolution, 4); err != nil {
		return
	}
	if err = WriteFixed(w, self.VorizontalResolution, 4); err != nil {
		return
	}
	if err = WriteDummy(w, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.FrameCount, 2); err != nil {
		return
	}
	if err = WriteString(w, self.CompressorName, 32); err != nil {
		return
	}
	if err = WriteInt(w, self.Depth, 2); err != nil {
		return
	}
	if err = WriteInt(w, self.ColorTableId, 2); err != nil {
		return
	}
	if self.Conf != nil {
		if err = WriteAvc1Conf(w, self.Conf); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkAvc1Desc(w Walker, self *Avc1Desc) {

	w.StartStruct("Avc1Desc")
	w.Name("DataRefIdx")
	w.Int(self.DataRefIdx)
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Revision")
	w.Int(self.Revision)
	w.Name("Vendor")
	w.Int(self.Vendor)
	w.Name("TemporalQuality")
	w.Int(self.TemporalQuality)
	w.Name("SpatialQuality")
	w.Int(self.SpatialQuality)
	w.Name("Width")
	w.Int(self.Width)
	w.Name("Height")
	w.Int(self.Height)
	w.Name("HorizontalResolution")
	w.Fixed(self.HorizontalResolution)
	w.Name("VorizontalResolution")
	w.Fixed(self.VorizontalResolution)
	w.Name("FrameCount")
	w.Int(self.FrameCount)
	w.Name("CompressorName")
	w.String(self.CompressorName)
	w.Name("Depth")
	w.Int(self.Depth)
	w.Name("ColorTableId")
	w.Int(self.ColorTableId)
	if self.Conf != nil {
		WalkAvc1Conf(w, self.Conf)
	}
	w.EndStruct()
	return
}

type Avc1Conf struct {
	Record AVCDecoderConfRecord
}

func ReadAvc1Conf(r *io.LimitedReader) (res *Avc1Conf, err error) {

	self := &Avc1Conf{}
	if self.Record, err = ReadAVCDecoderConfRecord(r); err != nil {
		return
	}
	res = self
	return
}
func WriteAvc1Conf(w io.WriteSeeker, self *Avc1Conf) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "avcC"); err != nil {
		return
	}
	w = aw
	if err = WriteAVCDecoderConfRecord(w, self.Record); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkAvc1Conf(w Walker, self *Avc1Conf) {

	w.StartStruct("Avc1Conf")
	WalkAVCDecoderConfRecord(w, self.Record)
	w.EndStruct()
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
func WriteTimeToSample(w io.WriteSeeker, self *TimeToSample) (err error) {

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
func WalkTimeToSample(w Walker, self *TimeToSample) {

	w.StartStruct("TimeToSample")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	for i, item := range self.Entries {
		if w.FilterArrayItem("TimeToSample", "Entries", i, len(self.Entries)) {
			WalkTimeToSampleEntry(w, item)
		} else {
			w.ArrayLeft(i, len(self.Entries))
			break
		}
	}
	w.EndStruct()
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
func WriteTimeToSampleEntry(w io.WriteSeeker, self TimeToSampleEntry) (err error) {

	if err = WriteInt(w, self.Count, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Duration, 4); err != nil {
		return
	}
	return
}
func WalkTimeToSampleEntry(w Walker, self TimeToSampleEntry) {

	w.StartStruct("TimeToSampleEntry")
	w.Name("Count")
	w.Int(self.Count)
	w.Name("Duration")
	w.Int(self.Duration)
	w.EndStruct()
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
func WriteSampleToChunk(w io.WriteSeeker, self *SampleToChunk) (err error) {

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
func WalkSampleToChunk(w Walker, self *SampleToChunk) {

	w.StartStruct("SampleToChunk")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	for i, item := range self.Entries {
		if w.FilterArrayItem("SampleToChunk", "Entries", i, len(self.Entries)) {
			WalkSampleToChunkEntry(w, item)
		} else {
			w.ArrayLeft(i, len(self.Entries))
			break
		}
	}
	w.EndStruct()
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
func WriteSampleToChunkEntry(w io.WriteSeeker, self SampleToChunkEntry) (err error) {

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
func WalkSampleToChunkEntry(w Walker, self SampleToChunkEntry) {

	w.StartStruct("SampleToChunkEntry")
	w.Name("FirstChunk")
	w.Int(self.FirstChunk)
	w.Name("SamplesPerChunk")
	w.Int(self.SamplesPerChunk)
	w.Name("SampleDescId")
	w.Int(self.SampleDescId)
	w.EndStruct()
	return
}

type CompositionOffset struct {
	Version int
	Flags   int
	Entries []CompositionOffsetEntry
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
func WalkCompositionOffset(w Walker, self *CompositionOffset) {

	w.StartStruct("CompositionOffset")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	for i, item := range self.Entries {
		if w.FilterArrayItem("CompositionOffset", "Entries", i, len(self.Entries)) {
			WalkCompositionOffsetEntry(w, item)
		} else {
			w.ArrayLeft(i, len(self.Entries))
			break
		}
	}
	w.EndStruct()
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
func WriteCompositionOffsetEntry(w io.WriteSeeker, self CompositionOffsetEntry) (err error) {

	if err = WriteInt(w, self.Count, 4); err != nil {
		return
	}
	if err = WriteInt(w, self.Offset, 4); err != nil {
		return
	}
	return
}
func WalkCompositionOffsetEntry(w Walker, self CompositionOffsetEntry) {

	w.StartStruct("CompositionOffsetEntry")
	w.Name("Count")
	w.Int(self.Count)
	w.Name("Offset")
	w.Int(self.Offset)
	w.EndStruct()
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
func WriteSyncSample(w io.WriteSeeker, self *SyncSample) (err error) {

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
func WalkSyncSample(w Walker, self *SyncSample) {

	w.StartStruct("SyncSample")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	for i, item := range self.Entries {
		if w.FilterArrayItem("SyncSample", "Entries", i, len(self.Entries)) {
			w.Name("Entries")
			w.Int(item)
		} else {
			w.ArrayLeft(i, len(self.Entries))
			break
		}
	}
	w.EndStruct()
	return
}

type SampleSize struct {
	Version    int
	Flags      int
	SampleSize int
	Entries    []int
}

func ReadSampleSize(r *io.LimitedReader) (res *SampleSize, err error) {

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
func WalkSampleSize(w Walker, self *SampleSize) {

	w.StartStruct("SampleSize")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("SampleSize")
	w.Int(self.SampleSize)
	for i, item := range self.Entries {
		if w.FilterArrayItem("SampleSize", "Entries", i, len(self.Entries)) {
			w.Name("Entries")
			w.Int(item)
		} else {
			w.ArrayLeft(i, len(self.Entries))
			break
		}
	}
	w.EndStruct()
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
func WriteChunkOffset(w io.WriteSeeker, self *ChunkOffset) (err error) {

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
func WalkChunkOffset(w Walker, self *ChunkOffset) {

	w.StartStruct("ChunkOffset")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	for i, item := range self.Entries {
		if w.FilterArrayItem("ChunkOffset", "Entries", i, len(self.Entries)) {
			w.Name("Entries")
			w.Int(item)
		} else {
			w.ArrayLeft(i, len(self.Entries))
			break
		}
	}
	w.EndStruct()
	return
}

type MovieFrag struct {
	Header *MovieFragHeader
	Tracks []*TrackFrag
}

func ReadMovieFrag(r *io.LimitedReader) (res *MovieFrag, err error) {

	self := &MovieFrag{}
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "mfhd":
			{
				if self.Header, err = ReadMovieFragHeader(ar); err != nil {
					return
				}
			}
		case "traf":
			{
				var item *TrackFrag
				if item, err = ReadTrackFrag(ar); err != nil {
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
func WriteMovieFrag(w io.WriteSeeker, self *MovieFrag) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "moof"); err != nil {
		return
	}
	w = aw
	if self.Header != nil {
		if err = WriteMovieFragHeader(w, self.Header); err != nil {
			return
		}
	}
	if self.Tracks != nil {
		for _, elem := range self.Tracks {
			if err = WriteTrackFrag(w, elem); err != nil {
				return
			}
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkMovieFrag(w Walker, self *MovieFrag) {

	w.StartStruct("MovieFrag")
	if self.Header != nil {
		WalkMovieFragHeader(w, self.Header)
	}
	for i, item := range self.Tracks {
		if w.FilterArrayItem("MovieFrag", "Tracks", i, len(self.Tracks)) {
			if item != nil {
				WalkTrackFrag(w, item)
			}
		} else {
			w.ArrayLeft(i, len(self.Tracks))
			break
		}
	}
	w.EndStruct()
	return
}

type MovieFragHeader struct {
	Version int
	Flags   int
	SeqNum  int
}

func ReadMovieFragHeader(r *io.LimitedReader) (res *MovieFragHeader, err error) {

	self := &MovieFragHeader{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.SeqNum, err = ReadInt(r, 4); err != nil {
		return
	}
	res = self
	return
}
func WriteMovieFragHeader(w io.WriteSeeker, self *MovieFragHeader) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "mfhd"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if err = WriteInt(w, self.SeqNum, 4); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkMovieFragHeader(w Walker, self *MovieFragHeader) {

	w.StartStruct("MovieFragHeader")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("SeqNum")
	w.Int(self.SeqNum)
	w.EndStruct()
	return
}

type TrackFrag struct {
	Header     *TrackFragHeader
	DecodeTime *TrackFragDecodeTime
	Run        *TrackFragRun
}

func ReadTrackFrag(r *io.LimitedReader) (res *TrackFrag, err error) {

	self := &TrackFrag{}
	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}
		switch cc4 {
		case "tfhd":
			{
				if self.Header, err = ReadTrackFragHeader(ar); err != nil {
					return
				}
			}
		case "tfdt":
			{
				if self.DecodeTime, err = ReadTrackFragDecodeTime(ar); err != nil {
					return
				}
			}
		case "trun":
			{
				if self.Run, err = ReadTrackFragRun(ar); err != nil {
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
func WriteTrackFrag(w io.WriteSeeker, self *TrackFrag) (err error) {

	var aw *Writer
	if aw, err = WriteAtomHeader(w, "traf"); err != nil {
		return
	}
	w = aw
	if self.Header != nil {
		if err = WriteTrackFragHeader(w, self.Header); err != nil {
			return
		}
	}
	if self.DecodeTime != nil {
		if err = WriteTrackFragDecodeTime(w, self.DecodeTime); err != nil {
			return
		}
	}
	if self.Run != nil {
		if err = WriteTrackFragRun(w, self.Run); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}
func WalkTrackFrag(w Walker, self *TrackFrag) {

	w.StartStruct("TrackFrag")
	if self.Header != nil {
		WalkTrackFragHeader(w, self.Header)
	}
	if self.DecodeTime != nil {
		WalkTrackFragDecodeTime(w, self.DecodeTime)
	}
	if self.Run != nil {
		WalkTrackFragRun(w, self.Run)
	}
	w.EndStruct()
	return
}
