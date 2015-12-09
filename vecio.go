
package ts

import (
	"io"
)

type iovec struct {
	data [][]byte
	Len int
}

func (self *iovec) Append(b []byte) {
	self.data = append(self.data, b)
	self.Len += len(b)
}

func (self *iovec) WriteTo(w io.Writer, n int) (written int, err error) {
	for n > 0 && self.Len > 0 {
		data := self.data[0]

		var b []byte
		if n > len(data) {
			b = data
		} else {
			b = data[:n]
		}

		data = data[len(b):]
		if len(data) == 0 {
			self.data = self.data[1:]
		} else {
			self.data[0] = data
		}
		self.Len -= len(b)
		n -= len(b)
		written += len(b)

		if _, err = w.Write(b); err != nil {
			return
		}
	}
	return
}

