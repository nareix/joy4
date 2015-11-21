
package atom

import (
	"io"
	"io/ioutil"
	"log"
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

func ReadAtomHeader(r io.Reader, targetCC4 string) (res *io.LimitedReader, cc4 string, err error) {
	for {
		var size int
		if size, err = ReadInt(r, 4); err != nil {
			return
		}
		if size == 0 {
			continue
		}

		if cc4, err = ReadString(r, 4); err != nil {
			return
		}
		size = size - 8

		if false {
			log.Println(cc4, targetCC4, size, cc4 == targetCC4)
		}

		if targetCC4 != "" && cc4 != targetCC4 {
			log.Println("ReadAtomHeader skip:", cc4)
			if _, err = ReadDummy(r, size); err != nil {
				return
			}
			continue
		}

		res = &io.LimitedReader{
			R: r,
			N: int64(size),
		}
		return
	}
}

