
package ts

import (
	"io"
	"io/ioutil"
	"fmt"
)

type TSHeader struct {
	PID uint
	PCR uint64
	OPCR uint64
	ContinuityCounter uint
	PayloadUnitStart bool
}

func ReadUInt(r io.Reader, n int) (res uint, err error) {
	var b [4]byte
	if _, err = r.Read(b[0:n]); err != nil {
		return
	}
	for i := 0; i < n; i++ {
		res <<= 8
		res |= uint(b[i])
	}
	return
}

func ReadDummy(r io.Reader, n int) (err error) {
	_, err = io.CopyN(ioutil.Discard, r, int64(n))
	return
}

func ReadUInt64(r io.Reader, n int) (res uint64, err error) {
	var res32 uint
	if n > 4 {
		if res32, err = ReadUInt(r, n-4); err != nil {
			return
		}
		res |= uint64(res32)<<(uint(n-4)*8)
		n = 4
	}
	if res32, err = ReadUInt(r, n); err != nil {
		return
	}
	res |= uint64(res32)
	return
}

func ReadTSPacket(r io.Reader, data []byte) (self TSHeader, n int, err error) {
	lr := &io.LimitedReader{R: r, N: 188}
	if self, err = ReadTSHeader(lr); err != nil {
		return
	}
	if n, err = lr.Read(data[:lr.N]); err != nil {
		return
	}
	return
}

const (
	ElementaryStreamTypeH264 = 0x1B
	ElementaryStreamTypeAdtsAAC = 0x0F
)

type PATEntry struct {
	ProgramNumber uint
	NetworkPID uint
	ProgramMapPID uint
}

type PAT struct {
	Entries []PATEntry
}

type PMT struct {
	PCRPID uint
	ProgramDescriptors []Descriptor
	ElementaryStreamInfos []ElementaryStreamInfo
}

type Descriptor struct {
	Tag uint
	Data []byte
}

type ElementaryStreamInfo struct {
	StreamType uint
	ElementaryPID uint
	Descriptors []Descriptor
}

type PSI struct {
	TableIdExtension uint
	TableId uint
}

func ReadPSI(r io.Reader) (self PSI, lr *io.LimitedReader, err error) {
	var flags, pointer, length uint

	// pointer field
	if pointer, err = ReadUInt(r, 1); err != nil {
		return
	}
	if pointer != 0 {
		if err = ReadDummy(r, int(pointer)); err != nil {
			return
		}
	}

	// table_id
	if self.TableId, err = ReadUInt(r, 1); err != nil {
		return
	}

	// reserved(4)
	// section_length(10)
	if flags, err = ReadUInt(r, 2); err != nil {
		return
	}
	length = flags & 0x3FF
	lr = &io.LimitedReader{R: r, N: int64(length)}

	// Table ID extension(16)
	if _, err = ReadUInt(lr, 2); err != nil {
		return
	}

	// _(2)
	// version(5)
	// Current_next_indicator(1)
	if _, err = ReadUInt(lr, 1); err != nil {
		return
	}

	// section_number(8)
	if _, err = ReadUInt(lr, 1); err != nil {
		return
	}

	// last_section_number(8)
	if _, err = ReadUInt(lr, 1); err != nil {
		return
	}

	lr.N -= 4
	return
}

func ReadPMT(r io.Reader) (self PMT, err error) {
	var lr *io.LimitedReader
	//var psi PSI

	if _, lr, err = ReadPSI(r); err != nil {
		return
	}

	var flags, length uint

	// Reserved(3)
	// PCRPID(13)
	if flags, err = ReadUInt(lr, 2); err != nil {
		return
	}
	self.PCRPID = flags & 0x1fff

	// Reserved(6)
	// Program info length(10)
	if flags, err = ReadUInt(lr, 2); err != nil {
		return
	}
	length = flags & 0x3ff

	readDescs := func(lr *io.LimitedReader) (res []Descriptor, err error) {
		var desc Descriptor
		for lr.N > 0 {
			if desc.Tag, err = ReadUInt(lr, 1); err != nil {
				return
			}
			var length uint
			if length, err = ReadUInt(lr, 1); err != nil {
				return
			}
			desc.Data = make([]byte, length)
			if _, err = lr.Read(desc.Data); err != nil {
				return
			}
			res = append(res, desc)
		}
		return
	}

	if length > 0 {
		lr := &io.LimitedReader{R: lr, N: int64(length)}
		if self.ProgramDescriptors, err = readDescs(lr); err != nil {
			return
		}
	}

	for lr.N > 0 {
		var info ElementaryStreamInfo
		if info.StreamType, err = ReadUInt(lr, 1); err != nil {
			return
		}

		// Reserved(3)
		// Elementary PID(13)
		if flags, err = ReadUInt(lr, 2); err != nil {
			return
		}
		info.ElementaryPID = flags & 0x1fff

		// Reserved(6)
		// ES Info length length(10)
		if flags, err = ReadUInt(lr, 2); err != nil {
			return
		}
		length = flags & 0x3ff

		if length > 0 {
			lr := &io.LimitedReader{R: lr, N: int64(length)}
			if info.Descriptors, err = readDescs(lr); err != nil {
				return
			}
		}
		self.ElementaryStreamInfos = append(self.ElementaryStreamInfos, info)
	}

	return
}

func ReadPAT(r io.Reader) (self PAT, err error) {
	var lr *io.LimitedReader
	//var psi PSI

	if _, lr, err = ReadPSI(r); err != nil {
		return
	}

	for lr.N > 0 {
		entry := PATEntry{}
		if entry.ProgramNumber, err = ReadUInt(lr, 2); err != nil {
			return
		}
		if entry.ProgramNumber == 0 {
			if entry.NetworkPID, err = ReadUInt(lr, 2); err != nil {
				return
			}
			entry.NetworkPID &= 0x1fff
		} else {
			if entry.ProgramMapPID, err = ReadUInt(lr, 2); err != nil {
				return
			}
			entry.ProgramMapPID &= 0x1fff
		}
		self.Entries = append(self.Entries, entry)
	}

	return
}

func ReadTSHeader(r io.Reader) (self TSHeader, err error) {
	var flags uint

	// sync(8)
	// transport_error_indicator(1)
	// payload_unit_start_indicator(1)
	// transport_priority(1)
	// pid(13)
	// Scrambling control(2)
	// Adaptation field flag(1)
	// Continuity counter(4)
	if flags, err = ReadUInt(r, 4); err != nil {
		return
	}

	if flags & 0x400000 != 0 {
		self.PayloadUnitStart = true
	}

	if (flags & 0xff000000) >> 24 != 0x47 {
		err = fmt.Errorf("invalid sync")
		return
	}

	self.PID = (flags & 0x1fff00) >> 8
	self.ContinuityCounter = flags & 0xf

	if flags & 0x20 != 0 {
		var flags, length uint
		if length, err = ReadUInt(r, 1); err != nil {
			return
		}
		lr := &io.LimitedReader{R: r, N: int64(length)}
		if flags, err = ReadUInt(lr, 1); err != nil {
			return
		}

		// PCR
		if flags & 0x10 != 0 {
			if self.PCR, err = ReadUInt64(lr, 6); err != nil {
				return
			}
		}

		// OPCR
		if flags & 0x08 != 0 {
			if self.OPCR, err = ReadUInt64(lr, 6); err != nil {
				return
			}
		}

		// Splice countdown
		if flags & 0x04 != 0 {
			if _, err = ReadUInt(lr, 1); err != nil {
				return
			}
		}

		// Transport private data
		if flags & 0x02 != 0 {
			var length uint
			if length, err = ReadUInt(lr, 1); err != nil {
				return
			}

			b := make([]byte, length)
			if _, err = lr.Read(b); err != nil {
				return
			}
		}

		// Adaptation extension
		if err = ReadDummy(lr, int(lr.N)); err != nil {
			return
		}
	}

	return
}

type PESHeader struct {
	StreamId uint // H264=0xe0 AAC=0xc0
	DataLength uint
	PTS uint64
	DTS uint64
	ESCR uint64
}

func PESUIntToTs(v uint64) (ts uint64) {
	// 0010	PTS 32..30 1	PTS 29..15 1 PTS 14..00 1
	return (((v>>33)&0x7)<<30) | (((v>>17)&0xef)<<15) | ((v>>1)&0xef)
}

func ReadPESHeader(r io.Reader) (res *PESHeader, err error) {
	var flags, length uint
	self := &PESHeader{}

	// start code 000001
	if flags, err = ReadUInt(r, 3); err != nil {
		return
	}
	if flags != 0x000001 {
		err = fmt.Errorf("invalid PES header")
		return
	}

	if self.StreamId, err = ReadUInt(r, 1); err != nil {
		return
	}

	if length, err = ReadUInt(r, 2); err != nil {
		return
	}
	lrAll := &io.LimitedReader{R: r, N: int64(length)}
	lr := lrAll

	// PES scrambling control
	// PES priority
	// data alignment indicator
	// copyright
	// original or copy
	if _, err = ReadUInt(lr, 1); err != nil {
		return
	}

	// PTS DTS flags(2)
	// ESCR flag(1)
	// ES rate flag(1)
	// DSM trick mode flag(1)
	// additional copy info flag(1)
	// PES CRC flag(1)
	// PES extension flag(1)
	if flags, err = ReadUInt(lr, 1); err != nil {
		return
	}

	// PES header data length(8)
	if length, err = ReadUInt(lr, 1); err != nil {
		return
	}
	lr = &io.LimitedReader{R: lr, N: int64(length)}

	if flags & 0x80 != 0 {
		var v uint64
		if v, err = ReadUInt64(lr, 5); err != nil {
			return
		}
		self.PTS = PESUIntToTs(v)
	}

	if flags & 0x40 != 0 && flags & 0x80 != 0 {
		var v uint64
		if v, err = ReadUInt64(lr, 5); err != nil {
			return
		}
		self.DTS = PESUIntToTs(v)
	}

	// ESCR flag
	if flags & 0x20 != 0 {
		if _, err = ReadUInt64(lr, 6); err != nil {
			return
		}
	}

	// ES rate flag
	if flags & 0x10 != 0 {
		if _, err = ReadUInt64(lr, 3); err != nil {
			return
		}
	}

	// additional copy info flag
	if flags & 0x04 != 0 {
		if _, err = ReadUInt(lr, 1); err != nil {
			return
		}
	}

	// PES CRC flag
	if flags & 0x02 != 0 {
		if _, err = ReadUInt(lr, 2); err != nil {
			return
		}
	}

	// PES extension flag
	if flags & 0x01 != 0 {
		var flags uint

		// PES private data flag(1)
		// pack header field flag(1)
		// program packet sequence counter flag(1)
		// P-STD buffer flag(1)
		// 111(3)
		// PES extension flag 2(1)
		if flags, err = ReadUInt(lr, 1); err != nil {
			return
		}

		// PES private data flag(1)
		if flags & 0x80 != 0 {
			// if set to 1 16 bytes of user defined data is appended to the header data field
			if err = ReadDummy(lr, 16); err != nil {
				return
			}
		}

		// pack header field flag(1)
		if flags & 0x40 != 0 {
			// if set to 1 the 8-bit pack field length value is appended to the header data field
			if err = ReadDummy(lr, 1); err != nil {
				return
			}
		}

		// program packet sequence counter flag(1)
		if flags & 0x20 != 0 {
			if err = ReadDummy(lr, 2); err != nil {
				return
			}
		}

		// P-STD buffer flag(1)
		if flags & 0x10 != 0 {
			if err = ReadDummy(lr, 2); err != nil {
				return
			}
		}

		// PES extension flag 2(1)
		if flags & 0x01 != 0 {
			if err = ReadDummy(lr, 2); err != nil {
				return
			}
		}
	}

	if lr.N > 0 {
		if err = ReadDummy(lr, int(lr.N)); err != nil {
			return
		}
	}

	self.DataLength = uint(lrAll.N)

	res = self
	return
}

