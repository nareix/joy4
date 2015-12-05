
package ts

import (
	_ "fmt"
	"io"
)

type TSWriter struct {
	W io.Writer
	ContinuityCounter uint
	PayloadUnitStart bool
}

func WriteUInt64(w io.Writer, val uint64, n int) (err error) {
	var b [8]byte
	for i := n-1; i >= 0; i-- {
		b[i] = byte(val)
		val >>= 8
	}
	if _, err = w.Write(b[:]); err != nil {
		return
	}
	return
}

func WriteUInt(w io.Writer, val uint, n int) (err error) {
	return WriteUInt64(w, uint64(val), n)
}

func WriteTSHeader(w io.Writer, self TSHeader) (err error) {
	var flags, extFlags uint

	// sync(8)
	// transport_error_indicator(1)
	// payload_unit_start_indicator(1)
	// transport_priority(1)
	// pid(13)
	// Scrambling control(2)
	// Adaptation field flag(1) 0x20
	// Payload flag(1) 0x10
	// Continuity counter(4)

	flags = 0x47<<24
	flags |= 0x10
	if self.PayloadUnitStart {
		flags |= 0x400000
	}
	flags |= (self.PID&0x1fff00)<<8
	flags |= self.ContinuityCounter&0xf

	if self.PCR != 0 {
		extFlags |= 0x20
	}
	if self.OPCR != 0 {
		extFlags |= 0x08
	}
	if self.RandomAccessIndicator {
		extFlags |= 0x40
	}

	if extFlags != 0 {
		flags |= 0x20
	}

	if err = WriteUInt(w, flags, 4); err != nil {
		return
	}

	if flags & 0x20 != 0 {
		var length uint

		// Discontinuity indicator	1	0x80	
		// Random Access indicator	1	0x40	
		// Elementary stream priority indicator	1	0x20	
		// PCR flag	1	0x10
		// OPCR flag	1	0x08	

		length = 1
		if extFlags & 0x10 != 0 {
			length += 6
		}
		if extFlags & 0x08 != 0 {
			length += 6
		}

		if err = WriteUInt(w, length, 1); err != nil {
			return
		}
		if err = WriteUInt(w, extFlags, 1); err != nil {
			return
		}

		if extFlags & 0x10 != 0 {
			if err = WriteUInt64(w, PCRToUInt(self.PCR), 6); err != nil {
				return
			}
		}

		if extFlags & 0x08 != 0 {
			if err = WriteUInt64(w, PCRToUInt(self.OPCR), 6); err != nil {
				return
			}
		}
	}

	return
}

type PSIWriter struct {
	W *TSWriter
}

func (self PSIWriter) Write(b []byte) (err error) {
	return
}

func (self PSIWriter) Finish() (err error) {
	return
}

type PESWriter struct {
	W io.Writer
}

type SimpleH264Writer struct {
	W io.Writer
	headerHasWritten bool
}

func WritePAT(w io.Writer, self PAT) (err error) {
	return
}

func (self *SimpleH264Writer) WriteSample(data []byte) (err error) {
	return
}

func (self *SimpleH264Writer) WriteNALU(data []byte) (err error) {
	return
}

