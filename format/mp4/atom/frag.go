package atom

import (
	_ "bytes"
	"fmt"
	"github.com/nareix/bits"
	"io"
)

const (
	TFHD_BASE_DATA_OFFSET     = 0x01
	TFHD_STSD_ID              = 0x02
	TFHD_DEFAULT_DURATION     = 0x08
	TFHD_DEFAULT_SIZE         = 0x10
	TFHD_DEFAULT_FLAGS        = 0x20
	TFHD_DURATION_IS_EMPTY    = 0x010000
	TFHD_DEFAULT_BASE_IS_MOOF = 0x020000
)

type TrackFragHeader struct {
	Version         int
	Flags           int
	Id              int
	DefaultSize     int
	DefaultDuration int
	DefaultFlags    int
	BaseDataOffset  int64
	StsdId          int
}

func WalkTrackFragHeader(w Walker, self *TrackFragHeader) {
	w.StartStruct("TrackFragHeader")
	w.Name("Flags")
	w.HexInt(self.Flags)
	w.Name("Id")
	w.Int(self.Id)
	w.Name("DefaultDuration")
	w.Int(self.DefaultDuration)
	w.Name("DefaultSize")
	w.Int(self.DefaultSize)
	w.Name("DefaultFlags")
	w.HexInt(self.DefaultFlags)
	w.EndStruct()
}

func WriteTrackFragHeader(w io.WriteSeeker, self *TrackFragHeader) (err error) {
	panic("unimplmented")
	return
}

func ReadTrackFragHeader(r *io.LimitedReader) (res *TrackFragHeader, err error) {
	self := &TrackFragHeader{}

	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.Id, err = ReadInt(r, 4); err != nil {
		return
	}

	if self.Flags&TFHD_BASE_DATA_OFFSET != 0 {
		if self.BaseDataOffset, err = bits.ReadInt64BE(r, 64); err != nil {
			return
		}
	}

	if self.Flags&TFHD_STSD_ID != 0 {
		if self.StsdId, err = ReadInt(r, 4); err != nil {
			return
		}
	}

	if self.Flags&TFHD_DEFAULT_DURATION != 0 {
		if self.DefaultDuration, err = ReadInt(r, 4); err != nil {
			return
		}
	}

	if self.Flags&TFHD_DEFAULT_SIZE != 0 {
		if self.DefaultSize, err = ReadInt(r, 4); err != nil {
			return
		}
	}

	if self.Flags&TFHD_DEFAULT_FLAGS != 0 {
		if self.DefaultFlags, err = ReadInt(r, 4); err != nil {
			return
		}
	}

	res = self
	return
}

const (
	TRUN_DATA_OFFSET        = 0x01
	TRUN_FIRST_SAMPLE_FLAGS = 0x04
	TRUN_SAMPLE_DURATION    = 0x100
	TRUN_SAMPLE_SIZE        = 0x200
	TRUN_SAMPLE_FLAGS       = 0x400
	TRUN_SAMPLE_CTS         = 0x800
)

type TrackFragRunEntry struct {
	Duration int
	Size     int
	Flags    int
	Cts      int
}

type TrackFragRun struct {
	Version          int
	Flags            int
	FirstSampleFlags int
	DataOffset       int
	Entries          []TrackFragRunEntry
}

func WalkTrackFragRun(w Walker, self *TrackFragRun) {
	w.StartStruct("TrackFragRun")
	w.Name("Flags")
	w.HexInt(self.Flags)
	w.Name("FirstSampleFlags")
	w.HexInt(self.FirstSampleFlags)
	w.Name("DataOffset")
	w.Int(self.DataOffset)
	w.Name("EntriesCount")
	w.Int(len(self.Entries))
	for i := 0; i < 10 && i < len(self.Entries); i++ {
		entry := self.Entries[i]
		w.Println(fmt.Sprintf("Entry[%d] Flags=%x Duration=%d Size=%d Cts=%d",
			i, entry.Flags, entry.Duration, entry.Size, entry.Cts))
	}
	w.EndStruct()
}

func WriteTrackFragRun(w io.WriteSeeker, self *TrackFragRun) (err error) {
	panic("unimplmented")
	return
}

func ReadTrackFragRun(r *io.LimitedReader) (res *TrackFragRun, err error) {
	self := &TrackFragRun{}

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

	if self.Flags&TRUN_DATA_OFFSET != 0 {
		if self.DataOffset, err = ReadInt(r, 4); err != nil {
			return
		}
	}
	if self.Flags&TRUN_FIRST_SAMPLE_FLAGS != 0 {
		if self.FirstSampleFlags, err = ReadInt(r, 4); err != nil {
			return
		}
	}

	for i := 0; i < count; i++ {
		var flags int

		if i > 0 {
			flags = self.Flags
		} else {
			flags = self.FirstSampleFlags
		}

		entry := TrackFragRunEntry{}
		if flags&TRUN_SAMPLE_DURATION != 0 {
			if entry.Duration, err = ReadInt(r, 4); err != nil {
				return
			}
		}
		if flags&TRUN_SAMPLE_SIZE != 0 {
			if entry.Size, err = ReadInt(r, 4); err != nil {
				return
			}
		}
		if flags&TRUN_SAMPLE_FLAGS != 0 {
			if entry.Flags, err = ReadInt(r, 4); err != nil {
				return
			}
		}
		if flags&TRUN_SAMPLE_CTS != 0 {
			if entry.Cts, err = ReadInt(r, 4); err != nil {
				return
			}
		}

		self.Entries = append(self.Entries, entry)
	}

	res = self
	return
}

type TrackFragDecodeTime struct {
	Version int
	Flags   int
	Time    int64
}

func ReadTrackFragDecodeTime(r *io.LimitedReader) (res *TrackFragDecodeTime, err error) {

	self := &TrackFragDecodeTime{}
	if self.Version, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.Version != 0 {
		if self.Time, err = bits.ReadInt64BE(r, 64); err != nil {
			return
		}
	} else {
		if self.Time, err = bits.ReadInt64BE(r, 32); err != nil {
			return
		}
	}
	res = self
	return
}

func WriteTrackFragDecodeTime(w io.WriteSeeker, self *TrackFragDecodeTime) (err error) {
	var aw *Writer
	if aw, err = WriteAtomHeader(w, "tfdt"); err != nil {
		return
	}
	w = aw
	if err = WriteInt(w, self.Version, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 3); err != nil {
		return
	}
	if self.Version != 0 {
		if err = bits.WriteInt64BE(w, self.Time, 64); err != nil {
			return
		}
	} else {
		if err = bits.WriteInt64BE(w, self.Time, 32); err != nil {
			return
		}
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

func WalkTrackFragDecodeTime(w Walker, self *TrackFragDecodeTime) {
	w.StartStruct("TrackFragDecodeTime")
	w.Name("Version")
	w.Int(self.Version)
	w.Name("Flags")
	w.Int(self.Flags)
	w.Name("Time")
	w.Int64(self.Time)
	w.EndStruct()
	return
}
