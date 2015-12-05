
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

func WritePSI(w io.Writer, self PSI, data []byte) (err error) {
	// pointer(8)
	// table_id(8)
	// reserved(4)=0xb,section_length(10)
	// Table ID extension(16)
	// resverd(2)=3,version(5),Current_next_indicator(1)
	// section_number(8)
	// last_section_number(8)
	// data
	// crc(32)

	// pointer(8)
	if err = WriteUInt(w, 0, 1); err != nil {
		return
	}

	cw := &Crc32Writer{W: w, Crc32: 0xffffffff}

	// table_id(8)
	if err = WriteUInt(cw, self.TableId, 1); err != nil {
		return
	}

	// reserved(4)=0xb,section_length(10)
	var flags, length uint
	length = 2+3+4+uint(len(data))
	flags = 0xb<<10|length
	if err = WriteUInt(cw, flags, 1); err != nil {
		return
	}

	// Table ID extension(16)
	if err = WriteUInt(cw, self.TableIdExtension, 2); err != nil {
		return
	}

	// resverd(2)=3,version(5)=0,Current_next_indicator(1)=1
	flags = 0x3<<6|1
	if err = WriteUInt(cw, flags, 1); err != nil {
		return
	}

	// section_number(8)
	if err = WriteUInt(cw, self.SecNum, 1); err != nil {
		return
	}

	// last_section_number(8)
	if err = WriteUInt(cw, self.LastSecNum, 1); err != nil {
		return
	}

	// data
	if _, err = cw.Write(data); err != nil {
		return
	}

	// crc(32)
	if err = WriteUInt(w, uint(cw.Crc32), 4); err != nil {
		return
	}

	return
}

func WritePES(w io.Writer, self PESHeader, data []byte) (err error) {
	// http://dvd.sourceforge.net/dvdinfo/pes-hdr.html

	var pts_dts_flags, header_length, packet_length uint

	// start code(24) 000001
	// StreamId(8)
	// packet_length(16)
	// resverd(6,2)=2,original_or_copy(0,1)=1
	// pts_dts_flags(6,2)
	// header_length(8)
	// pts(40)?
	// dts(40)?
	// data

	// start code(24) 000001
	if err = WriteUInt(w, 0x000001, 3); err != nil {
		return
	}

	// StreamId(8)
	if err = WriteUInt(w, self.StreamId, 1); err != nil {
		return
	}

	const PTS = 1<<7
	const DTS = 1<<6

	if self.PTS != 0 {
		pts_dts_flags |= PTS
		if self.DTS != 0 {
			pts_dts_flags |= DTS
		}
	}

	if pts_dts_flags & PTS != 0 {
		header_length += 5
	}
	if pts_dts_flags & DTS != 0 {
		header_length += 5
	}
	packet_length = 3+header_length+uint(len(data))

	// packet_length(16)
	if err = WriteUInt(w, packet_length, 2); err != nil {
		return
	}

	// resverd(6,2)=2,original_or_copy(0,1)=1
	if err = WriteUInt(w, 2<<6|1, 1); err != nil {
		return
	}

	// pts_dts_flags(6,2)
	if err = WriteUInt(w, pts_dts_flags, 1); err != nil {
		return
	}

	// header_length(8)
	if err = WriteUInt(w, header_length, 1); err != nil {
		return
	}

	// pts(40)?
	// dts(40)?
	if pts_dts_flags & PTS != 0 {
		if pts_dts_flags & DTS != 0 {
			if err = WriteUInt64(w, PESTsToUInt(self.PTS)|3<<36, 5); err != nil {
				return
			}
			if err = WriteUInt64(w, PESTsToUInt(self.DTS)|1<<36, 5); err != nil {
				return
			}
		} else {
			if err = WriteUInt64(w, PESTsToUInt(self.PTS)|2<<36, 5); err != nil {
				return
			}
		}
	}

	// data
	if _, err = w.Write(data); err != nil {
		return
	}

	return
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

