
package atom

import (
	"io"
)

type SampleDescEntry struct {
	Format string
	DataRefIndex int
	Data []byte
}

func ReadSampleDescEntry(r *io.LimitedReader) (res *SampleDescEntry, err error) {
	self := &SampleDescEntry{}
	if r, self.Format, err = ReadAtomHeader(r, ""); err != nil {
		return
	}
	if _, err = ReadDummy(r, 6); err != nil {
		return
	}
	if self.DataRefIndex, err = ReadInt(r, 2); err != nil {
		return
	}
	if self.Data, err = ReadBytes(r, int(r.N)); err != nil {
		return
	}
	res = self
	return
}

func WriteSampleDescEntry(w io.WriteSeeker, self *SampleDescEntry) (err error) {
	var aw *Writer
	if aw, err = WriteAtomHeader(w, self.Format); err != nil {
		return
	}
	w = aw
	if err = WriteDummy(w, 6); err != nil {
		return
	}
	if err = WriteInt(w, self.DataRefIndex, 2); err != nil {
		return
	}
	if err = WriteBytes(w, self.Data); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

type HandlerRefer struct {
	Version int
	Flags int
	Type string
	SubType string
	Name string
}

func ReadHandlerRefer(r *io.LimitedReader) (res *HandlerRefer, err error) {
	self := &HandlerRefer{}
	if self.Version, err = ReadInt(r, 3); err != nil {
		return
	}
	if self.Flags, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.Type, err = ReadString(r, 4); err != nil {
		return
	}
	if self.SubType, err = ReadString(r, 4); err != nil {
		return
	}
	if _, err = ReadDummy(r, 12); err != nil {
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
	if err = WriteInt(w, self.Version, 3); err != nil {
		return
	}
	if err = WriteInt(w, self.Flags, 1); err != nil {
		return
	}
	if err = WriteString(w, self.Type); err != nil {
		return
	}
	if err = WriteString(w, self.SubType); err != nil {
		return
	}
	if err = WriteDummy(w, 12); err != nil {
		return
	}
	if err = WriteString(w, self.Name); err != nil {
		return
	}
	if err = aw.Close(); err != nil {
		return
	}
	return
}

