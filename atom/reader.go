
package atom

import (
	"io"
	"io/ioutil"
)

type Fixed32 uint32
type TimeStamp uint32

func ReadBytes(r io.Reader, n int) (res []byte, err error) {
	res = make([]byte, n)
	if n, err = r.Read(res); err != nil {
		return
	}
	return
}

func ReadBytesLeft(r *io.LimitedReader) (res []byte, err error) {
	return ReadBytes(r, int(r.N))
}

func ReadUInt(r io.Reader, n int) (res uint, err error) {
	var b []byte
	if b, err = ReadBytes(r, n); err != nil {
		return
	}
	for i := 0; i < n; i++ {
		res <<= 8
		res += uint(b[i])
	}
	return
}

func ReadInt(r io.Reader, n int) (res int, err error) {
	var ui uint
	if ui, err = ReadUInt(r, n); err != nil {
		return
	}
	res = int(ui)
	return
}

func ReadFixed32(r io.Reader, n int) (res Fixed32, err error) {
	var ui uint
	if ui, err = ReadUInt(r, n); err != nil {
		return
	}
	res = Fixed32(ui)
	return
}

func ReadTimeStamp(r io.Reader, n int) (res TimeStamp, err error) {
	var ui uint
	if ui, err = ReadUInt(r, n); err != nil {
		return
	}
	res = TimeStamp(ui)
	return
}

func ReadString(r io.Reader, n int) (res string, err error) {
	var b []byte
	if b, err = ReadBytes(r, n); err != nil {
		return
	}
	res = string(b)
	return
}

func ReadDummy(r io.Reader, n int) (res int, err error) {
	_, err = io.CopyN(ioutil.Discard, r, int64(n))
	return
}

/*
func (self Reader) ReadAtom(atom Atom) (res Atom, err error) {
	for {
		var size int
		if size, err = self.ReadInt(4); err != nil {
			return
		}
		if size == 0 {
			continue
		}

		var cc4 string
		if cc4, err = self.ReadString(4); err != nil {
			return
		}
		if atom.CC4() != cc4 {
			if err = self.Skip(size); err != nil {
				return
			}
			continue
		}

		reader := &io.LimitedReader{
			R: self.Reader,
			N: int64(size - 8),
		}
		if err = atom.Read(Reader{reader}); err != nil {
			return
		}
		if err = self.Skip(int(reader.N)); err != nil {
			return
		}

		res = atom
		return
	}
}
*/

