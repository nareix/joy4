
package atom

import (
	"io"
)

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
	if self.SampleSize != 0 {
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
	if self.SampleSize != 0 {
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


