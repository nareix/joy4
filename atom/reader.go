
package atom

import (
	"io"
	"io/ioutil"
)

type Reader struct {
	io.LimitedReader
}

type Fixed32 uint32

func (self Reader) ReadUInt(n int) (res uint, err error) {
	b := make([]byte, n)
	if n, err = self.Read(b); err != nil {
		return
	}
	for i := 0; i < n; i++ {
		res <<= 8
		res += uint(b[i])
	}
	return
}

func (self Reader) ReadInt(n int) (res int, err error) {
	var resu uint
	if resu, err = self.ReadUInt(n); err != nil {
		return
	}
	res = int(resu)
	return
}

func (self Reader) ReadString(n int) (res string, err error) {
	b := make([]byte, n)
	if n, err = self.Read(b); err != nil {
		return
	}
	res = string(b)
	return
}

func (self Reader) Skip(n int) (err error) {
	_, err = io.CopyN(ioutil.Discard, self.Reader, int64(n))
	return
}

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

