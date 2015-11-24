
package atom

import (
	"io"
	"log"
)

func WriteBytes(w io.Writer, b []byte, n int) (err error) {
	if len(b) < n {
		b = append(b, make([]byte, n-len(b))...)
	}
	_, err = w.Write(b[:n])
	return
}

func WriteUInt(w io.Writer, val uint, n int) (err error) {
	var b [8]byte
	for i := n-1; i >= 0; i-- {
		b[i] = byte(val)
		val >>= 8
	}
	return WriteBytes(w, b[:], n)
}

func WriteInt(w io.Writer, val int, n int) (err error) {
	var uval uint
	if val < 0 {
		uval = uint((1<<uint(n*8))+val)
	} else {
		uval = uint(val)
	}
	return WriteUInt(w, uval, n)
}

func WriteFixed(w io.Writer, val Fixed, n int) (err error) {
	var uval uint

	if n == 2 {
		uval = uint(val)>>8
	} else if n == 4 {
		uval = uint(val)
	} else {
		panic("only fixed32 and fixed16 is supported")
	}

	return WriteUInt(w, uval, n)
}

func WriteTimeStamp(w io.Writer, ts TimeStamp, n int) (err error) {
	return WriteUInt(w, uint(ts), n)
}

func WriteString(w io.Writer, val string, n int) (err error) {
	return WriteBytes(w, []byte(val), n)
}

func WriteDummy(w io.Writer, n int) (err error) {
	return WriteBytes(w, []byte{}, n)
}

type Writer struct {
	io.WriteSeeker
	sizePos int64
}

func WriteEmptyInt(w io.WriteSeeker, n int) (pos int64, err error) {
	if pos, err = w.Seek(0, 1); err != nil {
		return
	}
	if err = WriteInt(w, 0, n); err != nil {
		return
	}
	return
}

func RefillInt(w io.WriteSeeker, pos int64, val int, n int) (err error) {
	var curPos int64
	if curPos, err = w.Seek(0, 1); err != nil {
		return
	}
	if _, err = w.Seek(pos, 0); err != nil {
		return
	}
	if err = WriteInt(w, val, n); err != nil {
		return
	}
	if _, err = w.Seek(curPos, 0); err != nil {
		return
	}
	return
}

func (self *Writer) Close() (err error) {
	var curPos int64
	if curPos, err = self.Seek(0, 1); err != nil {
		return
	}
	if err = RefillInt(self, self.sizePos, int(curPos - self.sizePos), 4); err != nil {
		return
	}
	if false {
		log.Println("writeback", self.sizePos, curPos, curPos-self.sizePos)
	}
	return
}

func WriteAtomHeader(w io.WriteSeeker, cc4 string) (res *Writer, err error) {
	self := &Writer{WriteSeeker: w}

	if self.sizePos, err = WriteEmptyInt(w, 4); err != nil {
		return
	}
	if err = WriteString(self, cc4, 4); err != nil {
		return
	}

	res = self
	return
}

