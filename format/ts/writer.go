package ts

import (
	"bytes"
	"io"
)

func WriteUInt64(w io.Writer, val uint64, n int) (err error) {
	var b [8]byte
	for i := n - 1; i >= 0; i-- {
		b[i] = byte(val)
		val >>= 8
	}
	if _, err = w.Write(b[:n]); err != nil {
		return
	}
	return
}

func WriteUInt(w io.Writer, val uint, n int) (err error) {
	return WriteUInt64(w, uint64(val), n)
}

func WritePSI(w io.Writer, self PSI, data []byte) (err error) {
	// pointer(8)
	// table_id(8)
	// reserved(4)=0xb,section_length(10)
	// Table ID extension(16)
	// section_syntax_indicator(1)=1,private_bit(1)=1,reserved(2)=3,unused(2)=0,section_length(10)
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

	// section_syntax_indicator(1)=1,private_bit(1)=0,reserved(2)=3,unused(2)=0,section_length(10)
	var flags, length uint
	length = 2 + 3 + 4 + uint(len(data))
	flags = 0xa<<12 | length
	if err = WriteUInt(cw, flags, 2); err != nil {
		return
	}

	// Table ID extension(16)
	if err = WriteUInt(cw, self.TableIdExtension, 2); err != nil {
		return
	}

	// resverd(2)=3,version(5)=0,Current_next_indicator(1)=1
	flags = 0x3<<6 | 1
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
	if err = WriteUInt(w, bswap32(uint(cw.Crc32)), 4); err != nil {
		return
	}

	return
}

func bswap32(v uint) uint {
	return (v >> 24) | ((v>>16)&0xff)<<8 | ((v>>8)&0xff)<<16 | (v&0xff)<<24
}

func WritePAT(w io.Writer, self PAT) (err error) {
	bw := &bytes.Buffer{}

	for _, entry := range self.Entries {
		if err = WriteUInt(bw, entry.ProgramNumber, 2); err != nil {
			return
		}
		if entry.ProgramNumber == 0 {
			if err = WriteUInt(bw, entry.NetworkPID&0x1fff|7<<13, 2); err != nil {
				return
			}
		} else {
			if err = WriteUInt(bw, entry.ProgramMapPID&0x1fff|7<<13, 2); err != nil {
				return
			}
		}
	}

	psi := PSI{
		TableIdExtension: 1,
	}
	if err = WritePSI(w, psi, bw.Bytes()); err != nil {
		return
	}

	return
}

func WritePMT(w io.Writer, self PMT) (err error) {
	writeDescs := func(w io.Writer, descs []Descriptor) (err error) {
		for _, desc := range descs {
			if err = WriteUInt(w, desc.Tag, 1); err != nil {
				return
			}
			if err = WriteUInt(w, uint(len(desc.Data)), 1); err != nil {
				return
			}
			if _, err = w.Write(desc.Data); err != nil {
				return
			}
		}
		return
	}

	writeBody := func(w io.Writer) (err error) {
		if err = WriteUInt(w, self.PCRPID|7<<13, 2); err != nil {
			return
		}

		bw := &bytes.Buffer{}
		if err = writeDescs(bw, self.ProgramDescriptors); err != nil {
			return
		}

		if err = WriteUInt(w, 0xf<<12|uint(bw.Len()), 2); err != nil {
			return
		}
		if _, err = w.Write(bw.Bytes()); err != nil {
			return
		}

		for _, info := range self.ElementaryStreamInfos {
			if err = WriteUInt(w, info.StreamType, 1); err != nil {
				return
			}

			// Reserved(3)
			// Elementary PID(13)
			if err = WriteUInt(w, info.ElementaryPID|7<<13, 2); err != nil {
				return
			}

			bw := &bytes.Buffer{}
			if err = writeDescs(bw, info.Descriptors); err != nil {
				return
			}

			// Reserved(6)
			// ES Info length length(10)
			if err = WriteUInt(w, uint(bw.Len())|0x3c<<10, 2); err != nil {
				return
			}

			if _, err = w.Write(bw.Bytes()); err != nil {
				return
			}
		}

		return
	}

	bw := &bytes.Buffer{}
	if err = writeBody(bw); err != nil {
		return
	}

	psi := PSI{
		TableId:          2,
		TableIdExtension: 1,
	}
	if err = WritePSI(w, psi, bw.Bytes()); err != nil {
		return
	}

	return
}

