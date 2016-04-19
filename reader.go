package ts

import (
	"fmt"
	"io"
	"io/ioutil"
)

var DebugReader = false

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
		res |= uint64(res32) << 32
		n = 4
	}
	if res32, err = ReadUInt(r, n); err != nil {
		return
	}
	res |= uint64(res32)
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

	if DebugReader {
		fmt.Printf("ts: flags %s\n", FieldsDumper{
			Fields: []struct {
				Length int
				Desc   string
			}{
				{8, "sync"},
				{1, "transport_error_indicator"},
				{1, "payload_unit_start_indicator"},
				{1, "transport_priority"},
				{13, "pid"},
				{2, "scrambling_control"},
				{1, "adaptation_field_flag"},
				{1, "payload_flag"},
				{4, "continuity_counter"},
			},
			Val:    flags,
			Length: 32,
		})
	}

	if flags&0x400000 != 0 {
		//  When set to '1' it indicates that this TS packet contains the first PES packet.
		self.PayloadUnitStart = true
	}

	if (flags&0xff000000)>>24 != 0x47 {
		err = fmt.Errorf("invalid sync")
		return
	}

	self.PID = (flags & 0x1fff00) >> 8
	self.ContinuityCounter = flags & 0xf

	if flags&0x20 != 0 {
		var flags, length uint
		if length, err = ReadUInt(r, 1); err != nil {
			return
		}
		if length > 0 {
			lr := &io.LimitedReader{R: r, N: int64(length)}
			if flags, err = ReadUInt(lr, 1); err != nil {
				return
			}

			if DebugReader {
				fmt.Printf("ts: ext_flags %s\n", FieldsDumper{
					Fields: []struct {
						Length int
						Desc   string
					}{
						{1, "discontinuity_indicator"},
						{1, "random_access_indicator"},
						{1, "elementary_stream_priority_indicator"},
						{1, "pcr_flag"},
						{1, "opcr_flag"},
						{1, "splicing_point_flag"},
						{1, "transport_private_data_flag"},
						{1, "adaptation_field_extension_flag"},
					},
					Val:    flags,
					Length: 8,
				})
			}

			// random_access_indicator
			if flags&0x40 != 0 {
				self.RandomAccessIndicator = true
			}

			// PCR
			if flags&0x10 != 0 {
				var v uint64
				if v, err = ReadUInt64(lr, 6); err != nil {
					return
				}
				// clock is 27MHz
				self.PCR = UIntToPCR(v)
				if DebugReader {
					fmt.Printf("ts: PCR %d %f\n", self.PCR, float64(self.PCR)/PCR_HZ)
				}
			}

			// OPCR
			if flags&0x08 != 0 {
				var v uint64
				if v, err = ReadUInt64(lr, 6); err != nil {
					return
				}
				self.OPCR = UIntToPCR(v)
			}

			// Splice countdown
			if flags&0x04 != 0 {
				if _, err = ReadUInt(lr, 1); err != nil {
					return
				}
			}

			// Transport private data
			if flags&0x02 != 0 {
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
			if lr.N > 0 {
				if DebugReader {
					// rubish
					fmt.Println("ts: skip", lr.N)
				}

				if err = ReadDummy(lr, int(lr.N)); err != nil {
					return
				}
			}
		}
	}

	return
}

func ReadTSPacket(r io.Reader, data []byte) (self TSHeader, n int, err error) {
	lr := &io.LimitedReader{R: r, N: 188}
	if self, err = ReadTSHeader(lr); err != nil {
		return
	}
	if DebugReader {
		fmt.Println("ts: data len", lr.N)
	}
	if n, err = lr.Read(data[:lr.N]); err != nil {
		return
	}
	return
}

func ReadPSI(r io.Reader) (self PSI, lr *io.LimitedReader, cr *Crc32Reader, err error) {
	var flags, pointer, length uint

	// pointer field
	if pointer, err = ReadUInt(r, 1); err != nil {
		return
	}

	if DebugReader {
		fmt.Printf("psi: pointer=%d\n", pointer)
	}

	if pointer != 0 {
		if err = ReadDummy(r, int(pointer)); err != nil {
			return
		}
	}

	cr = &Crc32Reader{R: r, Crc32: 0xffffffff}

	// table_id
	if self.TableId, err = ReadUInt(cr, 1); err != nil {
		return
	}

	// reserved(4)=0xb
	// section_length(10)
	if flags, err = ReadUInt(cr, 2); err != nil {
		return
	}
	length = flags & 0x3FF

	if DebugReader {
		fmt.Printf("psi: tableid=%d len=%d\n", self.TableId, length)
	}

	lr = &io.LimitedReader{R: cr, N: int64(length)}

	// Table ID extension(16)
	if self.TableIdExtension, err = ReadUInt(lr, 2); err != nil {
		return
	}

	// resverd(2)=3
	// version(5)
	// Current_next_indicator(1)
	if flags, err = ReadUInt(lr, 1); err != nil {
		return
	}

	if DebugReader {
		fmt.Printf("psi: %s\n", FieldsDumper{
			Fields: []struct {
				Length int
				Desc   string
			}{
				{2, "resverd"},
				{5, "version"},
				{1, "current_next_indicator"},
			},
			Val:    flags,
			Length: 8,
		})
	}

	// section_number(8)
	if self.SecNum, err = ReadUInt(lr, 1); err != nil {
		return
	}

	// last_section_number(8)
	if self.LastSecNum, err = ReadUInt(lr, 1); err != nil {
		return
	}

	if DebugReader {
		fmt.Printf("psi: table_id=%x table_extension=%x secnum=%x lastsecnum=%x\n",
			self.TableId,
			self.TableIdExtension,
			self.SecNum,
			self.LastSecNum,
		)
	}

	lr.N -= 4
	return
}

func ReadPMT(r io.Reader) (self PMT, err error) {
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

	var lr *io.LimitedReader
	var cr *Crc32Reader
	//var psi PSI

	if _, lr, cr, err = ReadPSI(r); err != nil {
		return
	}

	var flags, length uint

	// 111(3)
	// PCRPID(13)
	if flags, err = ReadUInt(lr, 2); err != nil {
		return
	}
	self.PCRPID = flags & 0x1fff

	if DebugReader {
		fmt.Printf("pmt: %s\n", FieldsDumper{
			Fields: []struct {
				Length int
				Desc   string
			}{
				{3, "reserved"},
				{13, "pcrpid"},
			},
			Val:    flags,
			Length: 16,
		})
	}

	// Reserved(4)=0xf
	// Reserved(2)=0x0
	// Program info length(10)
	if flags, err = ReadUInt(lr, 2); err != nil {
		return
	}
	length = flags & 0x3ff

	if DebugReader {
		fmt.Printf("pmt: ProgramDescriptorsLen=%d\n", length)
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

		if DebugReader {
			fmt.Printf("pmt: info1 %s\n", FieldsDumper{
				Fields: []struct {
					Length int
					Desc   string
				}{
					{3, "reserved"},
					{13, "elementary_pid"},
				},
				Val:    flags,
				Length: 16,
			})
		}

		// Reserved(6)
		// ES Info length(10)
		if flags, err = ReadUInt(lr, 2); err != nil {
			return
		}
		length = flags & 0x3ff

		if DebugReader {
			fmt.Printf("pmt: info2 %s\n", FieldsDumper{
				Fields: []struct {
					Length int
					Desc   string
				}{
					{6, "reserved"},
					{10, "es_info_length"},
				},
				Val:    flags,
				Length: 16,
			})
		}

		if length > 0 {
			lr := &io.LimitedReader{R: lr, N: int64(length)}
			if info.Descriptors, err = readDescs(lr); err != nil {
				return
			}
		}
		self.ElementaryStreamInfos = append(self.ElementaryStreamInfos, info)
	}

	if DebugReader {
		fmt.Printf("pmt: ProgramDescriptors %v\n", self.ProgramDescriptors)
		fmt.Printf("pmt: ElementaryStreamInfos %v\n", self.ElementaryStreamInfos)
	}

	if err = cr.ReadCrc32UIntAndCheck(); err != nil {
		if DebugReader {
			fmt.Printf("pmt: %s\n", err)
		}
		return
	}

	return
}

func ReadPAT(r io.Reader) (self PAT, err error) {
	var lr *io.LimitedReader
	var cr *Crc32Reader
	//var psi PSI

	if _, lr, cr, err = ReadPSI(r); err != nil {
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

	if err = cr.ReadCrc32UIntAndCheck(); err != nil {
		if DebugReader {
			fmt.Printf("pat: %s\n", err)
		}
		return
	}

	if DebugReader {
		fmt.Printf("pat: %v\n", self)
	}

	return
}

func ReadPESHeader(r io.Reader) (res *PESHeader, err error) {
	var flags, length uint
	self := &PESHeader{}

	// http://dvd.sourceforge.net/dvdinfo/pes-hdr.html

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
	if DebugReader {
		fmt.Printf("pes: StreamId=%x length=%d\n", self.StreamId, length)
	}

	if length == 0 {
		length = 1 << 31
	}
	lrAll := &io.LimitedReader{R: r, N: int64(length)}
	lr := lrAll

	// 10(2)
	// PES scrambling control(2)
	// PES priority(1)
	// data alignment indicator(1)
	// copyright(1)
	// original or copy(1)
	if _, err = ReadUInt(lr, 1); err != nil {
		return
	}

	if DebugReader {
		fmt.Printf("pes: %s\n", FieldsDumper{
			Fields: []struct {
				Length int
				Desc   string
			}{
				{2, "scrambling_control"},
				{1, "priority"},
				{1, "data_alignment_indicator"},
				{1, "copyright"},
				{1, "original_or_copy"},
			},
			Val:    flags,
			Length: 6,
		})
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

	if DebugReader {
		fmt.Printf("pes: %s\n", FieldsDumper{
			Fields: []struct {
				Length int
				Desc   string
			}{
				{2, "pts_dts_flags"},
				{1, "escr_flag"},
				{1, "es_rate_flag"},
				{1, "dsm_trick_mode_flag"},
				{1, "additional_copy_info_flag"},
				{1, "pes_crc_flag"},
				{1, "pes_extension_flag"},
			},
			Val:    flags,
			Length: 8,
		})
	}

	// PES header data length(8)
	if length, err = ReadUInt(lr, 1); err != nil {
		return
	}
	lr = &io.LimitedReader{R: lr, N: int64(length)}

	if flags&0x80 != 0 {
		var v uint64
		if v, err = ReadUInt64(lr, 5); err != nil {
			return
		}
		self.PTS = PESUIntToTs(v)

		if DebugReader {
			fmt.Printf("pes: pts %d %f\n", self.PTS, float64(self.PTS)/float64(PTS_HZ))
		}
	}

	if flags&0x40 != 0 && flags&0x80 != 0 {
		var v uint64
		if v, err = ReadUInt64(lr, 5); err != nil {
			return
		}
		self.DTS = PESUIntToTs(v)
		if DebugReader {
			fmt.Printf("pes: dts %d %f\n", self.DTS, float64(self.DTS)/float64(PTS_HZ))
		}
	}

	// ESCR flag
	if flags&0x20 != 0 {
		if _, err = ReadUInt64(lr, 6); err != nil {
			return
		}
	}

	// ES rate flag
	if flags&0x10 != 0 {
		if _, err = ReadUInt64(lr, 3); err != nil {
			return
		}
	}

	// additional copy info flag
	if flags&0x04 != 0 {
		if _, err = ReadUInt(lr, 1); err != nil {
			return
		}
	}

	// PES CRC flag
	if flags&0x02 != 0 {
		if _, err = ReadUInt(lr, 2); err != nil {
			return
		}
	}

	// PES extension flag
	if flags&0x01 != 0 {
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
		if flags&0x80 != 0 {
			// if set to 1 16 bytes of user defined data is appended to the header data field
			if err = ReadDummy(lr, 16); err != nil {
				return
			}
		}

		// pack header field flag(1)
		if flags&0x40 != 0 {
			// if set to 1 the 8-bit pack field length value is appended to the header data field
			if err = ReadDummy(lr, 1); err != nil {
				return
			}
		}

		// program packet sequence counter flag(1)
		if flags&0x20 != 0 {
			if err = ReadDummy(lr, 2); err != nil {
				return
			}
		}

		// P-STD buffer flag(1)
		if flags&0x10 != 0 {
			if err = ReadDummy(lr, 2); err != nil {
				return
			}
		}

		// PES extension flag 2(1)
		if flags&0x01 != 0 {
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

	if lrAll.N < 65536 {
		self.DataLength = uint(lrAll.N)
	}

	res = self
	return
}
