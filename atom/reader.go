
package atom

import (
	"io"
	"io/ioutil"
	"log"
)

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
	var uval uint
	if uval, err = ReadUInt(r, n); err != nil {
		return
	}
	if uval&(1<<uint(n*8-1)) != 0 {
		res = -int((1<<uint(n*8))-uval)
	} else {
		res = int(uval)
	}
	return
}

func ReadFixed(r io.Reader, n int) (res Fixed, err error) {
	var ui uint
	if ui, err = ReadUInt(r, n); err != nil {
		return
	}

	if n == 2 {
		res = Fixed(ui<<8)
	} else if n == 4 {
		res = Fixed(ui)
	} else {
		panic("only fixed32 and fixed16 is supported")
	}

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

