package ts

import (
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"unsafe"
)

type iovec struct {
	data [][]byte
	Len  int
	pos  int
	idx  int
}

func (self *iovec) Prepend(b []byte) {
	self.data = append([][]byte{b}, self.data...)
	self.Len += len(b)
}

func (self *iovec) Append(b []byte) {
	self.data = append(self.data, b)
	self.Len += len(b)
}

func (self *iovec) WriteTo(w io.Writer, n int) (written int, err error) {
	for n > 0 && self.Len > 0 {
		data := self.data[self.idx]

		var b []byte
		if n > len(data) {
			b = data
		} else {
			b = data[:n]
		}

		data = data[len(b):]
		if len(data) == 0 {
			self.idx++
		} else {
			self.data[self.idx] = data
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

type sysiovec struct {
	Base uintptr
	Len  uint64
}

type vecWriter struct {
	fd            uintptr
	smallBytesBuf []byte
	iov           []sysiovec
}

func (self *vecWriter) Write(p []byte) (written int, err error) {
	iov := sysiovec{
		Len: uint64(len(p)),
	}

	if len(p) < 16 {
		iov.Base = uintptr(len(self.smallBytesBuf))
		self.smallBytesBuf = append(self.smallBytesBuf, p...)
	} else {
		iov.Base = uintptr(unsafe.Pointer(&p[0]))
	}

	self.iov = append(self.iov, iov)
	return
}

func (self *vecWriter) Flush() (err error) {
	for i := range self.iov {
		iov := &self.iov[i]
		if iov.Base < uintptr(len(self.smallBytesBuf)) {
			iov.Base = uintptr(unsafe.Pointer(&self.smallBytesBuf[iov.Base]))
		}
	}

	N := 1024
	for i := 0; i < len(self.iov); i += N {
		n := len(self.iov) - i
		if n > N {
			n = N
		}
		_, _, errno := syscall.Syscall(syscall.SYS_WRITEV, self.fd, uintptr(unsafe.Pointer(&self.iov[i])), uintptr(n))
		if errno != 0 {
			err = fmt.Errorf("writev failed with error: %d", errno)
			return
		}
	}

	if DebugWriter {
		fmt.Printf("vecw: smallBytesBuf=%d iovNr=%d\n", len(self.smallBytesBuf), len(self.iov))
	}

	self.iov = self.iov[:0]
	self.smallBytesBuf = self.smallBytesBuf[:0]

	return
}

func newVecWriter(w io.Writer) (vecw *vecWriter) {
	var err error
	var f *os.File

	switch obj := w.(type) {
	case *net.TCPConn:
		f, err = obj.File()
		if err != nil {
			return
		}
	case *os.File:
		f = obj
	default:
		return
	}

	vecw = &vecWriter{
		fd: f.Fd(),
	}
	return
}
