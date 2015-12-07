
package ts

import (
	"io"
)

func getSeekerLength(data io.Seeker) (length int64) {
	length, _ = data.Seek(0, 2)
	data.Seek(0, 0)
	return
}

type multiReadSeeker struct {
	readers []io.ReadSeeker
}

func (mr *multiReadSeeker) Seek(offset int64, whence int) (n int64, err error) {
	if whence == 2 {
		for _, reader := range mr.readers {
			n += getSeekerLength(reader)
		}
	}
	return
}

func (mr *multiReadSeeker) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 {
		n, err = mr.readers[0].Read(p)
		if n > 0 || err != io.EOF {
			if err == io.EOF {
				err = nil
			}
			return
		}
		mr.readers = mr.readers[1:]
	}
	return 0, io.EOF
}

