package ts

import (
	"bytes"
	"fmt"
	"io"
)

var DebugWriter = false

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

func makeRepeatValBytes(val byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = val
	}
	return b
}

func WriteRepeatVal(w io.Writer, val byte, n int) (err error) {
	_, err = w.Write(makeRepeatValBytes(val, n))
	return
}

func WriteTSHeader(w io.Writer, self TSHeader, dataLength int) (written int, err error) {
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

	flags = 0x47 << 24
	flags |= 0x10
	if self.PayloadUnitStart {
		flags |= 0x400000
	}
	flags |= (self.PID & 0x1fff) << 8
	flags |= self.ContinuityCounter & 0xf

	if DebugWriter {
		fmt.Printf("tsw: pid=%x\n", self.PID)
	}

	const PCR = 0x10
	const OPCR = 0x08
	const EXT = 0x20

	if self.PCR != 0 {
		extFlags |= PCR
	}
	if self.OPCR != 0 {
		extFlags |= OPCR
	}
	if self.RandomAccessIndicator {
		extFlags |= 0x40
	}
	if self.DiscontinuityIndicator {
		extFlags |= 0x80
	}

	if extFlags != 0 {
		flags |= EXT
	}

	// need padding
	if dataLength < 184 {
		flags |= EXT
	}

	if err = WriteUInt(w, flags, 4); err != nil {
		return
	}
	written += 4

	if flags&EXT != 0 {
		var length uint

		// Discontinuity indicator	1	0x80
		// Random Access indicator	1	0x40
		// Elementary stream priority indicator	1	0x20
		// PCR flag	1	0x10
		// OPCR flag	1	0x08

		length = 1 // extFlags
		if extFlags&PCR != 0 {
			length += 6
		}
		if extFlags&OPCR != 0 {
			length += 6
		}

		paddingLength := 0
		// need padding
		if int(length)+5+dataLength < 188 {
			paddingLength = 188 - dataLength - 5 - int(length)
			length = 188 - uint(dataLength) - 5
		}

		if DebugWriter {
			fmt.Printf("tsw: header padding=%d\n", paddingLength)
		}

		if err = WriteUInt(w, length, 1); err != nil {
			return
		}
		if err = WriteUInt(w, extFlags, 1); err != nil {
			return
		}

		if extFlags&PCR != 0 {
			if err = WriteUInt64(w, PCRToUInt(self.PCR), 6); err != nil {
				return
			}
		}

		if extFlags&OPCR != 0 {
			if err = WriteUInt64(w, PCRToUInt(self.OPCR), 6); err != nil {
				return
			}
		}

		if paddingLength > 0 {
			if err = WriteRepeatVal(w, 0xff, paddingLength); err != nil {
				return
			}
		}

		written += int(length) + 1
	}

	return
}

type TSWriter struct {
	W   io.Writer
	PID uint
	TSHeader
	DisableHeaderPadding   bool
	DiscontinuityIndicator bool

	vecw *vecWriter
}

func (self *TSWriter) EnableVecWriter() {
	if self.vecw == nil {
		self.vecw = newVecWriter(self.W)

		if DebugWriter && self.vecw != nil {
			fmt.Println("tsw: enabled vec writer")
		}
	}
}

func (self *TSWriter) WriteIovec(data *iovec) (err error) {
	if self.vecw != nil {
		if err = self.WriteIovecTo(self.vecw, data); err != nil {
			return
		}
		if err = self.vecw.Flush(); err != nil {
			return
		}
	} else {
		if err = self.WriteIovecTo(self.W, data); err != nil {
			return
		}
	}
	return
}

func (self *TSWriter) WriteIovecTo(w io.Writer, data *iovec) (err error) {
	for i := 0; data.Len > 0; i++ {
		header := TSHeader{
			PID:                    self.PID,
			ContinuityCounter:      self.ContinuityCounter,
			DiscontinuityIndicator: self.DiscontinuityIndicator,
		}

		if i == 0 {
			header.PayloadUnitStart = true
			header.RandomAccessIndicator = self.RandomAccessIndicator
			header.PCR = self.PCR
			header.OPCR = self.OPCR
		}

		requestLength := data.Len
		if self.DisableHeaderPadding {
			requestLength = 188
		}
		var headerLength int
		if headerLength, err = WriteTSHeader(w, header, requestLength); err != nil {
			return
		}
		payloadLength := 188 - headerLength
		if self.DisableHeaderPadding && data.Len < payloadLength {
			data.Append(makeRepeatValBytes(0xff, payloadLength-data.Len))
		}

		if DebugWriter {
			fmt.Printf("tsw: payloadLength=%d dataLength=%d\n", payloadLength, data.Len)
		}

		if _, err = data.WriteTo(w, payloadLength); err != nil {
			return
		}

		self.ContinuityCounter++
	}
	return
}

func (self *TSWriter) WriteTo(w io.Writer, data []byte) (err error) {
	iov := &iovec{}
	iov.Append(data)
	return self.WriteIovecTo(w, iov)
}

func (self *TSWriter) Write(data []byte) (err error) {
	iov := &iovec{}
	iov.Append(data)
	return self.WriteIovec(iov)
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

	if DebugWriter {
		fmt.Printf("psiw: length=%d\n", length)
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

func WritePESHeader(w io.Writer, self PESHeader, dataLength int) (err error) {
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

	const PTS = 1 << 7
	const DTS = 1 << 6

	if self.PTS != 0 {
		pts_dts_flags |= PTS
		if self.DTS != 0 {
			pts_dts_flags |= DTS
		}
	}

	if pts_dts_flags&PTS != 0 {
		header_length += 5
	}
	if pts_dts_flags&DTS != 0 {
		header_length += 5
	}

	if dataLength > 0 {
		packet_length = uint(dataLength) + header_length + 3
	}
	// packet_length(16) if zero then variable length
	// Specifies the number of bytes remaining in the packet after this field. Can be zero.
	// If the PES packet length is set to zero, the PES packet can be of any length.
	// A value of zero for the PES packet length can be used only when the PES packet payload is a video elementary stream.
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
	if pts_dts_flags&PTS != 0 {
		if pts_dts_flags&DTS != 0 {
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

	return
}

func WritePESPacket(w *TSWriter, header PESHeader, data []byte) (err error) {
	bw := &bytes.Buffer{}
	if err = WritePESHeader(bw, header, len(data)); err != nil {
		return
	}
	iov := &iovec{}
	iov.Append(bw.Bytes())
	iov.Append(data)
	if err = w.WriteIovec(iov); err != nil {
		return
	}
	return
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

func WritePATPacket(w io.Writer, pat PAT) (err error) {
	tsw := &TSWriter{
		W:                    w,
		PID:                  0,
		DisableHeaderPadding: true,
	}
	bw := &bytes.Buffer{}
	if err = WritePAT(bw, pat); err != nil {
		return
	}
	if err = tsw.Write(bw.Bytes()); err != nil {
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

func WritePMTPacket(w io.Writer, pmt PMT, pid uint) (err error) {
	tsw := &TSWriter{
		W:                    w,
		PID:                  pid,
		DisableHeaderPadding: true,
	}
	bw := &bytes.Buffer{}
	if err = WritePMT(bw, pmt); err != nil {
		return
	}
	if err = tsw.Write(bw.Bytes()); err != nil {
		return
	}
	return
}

type SimpleH264Writer struct {
	W         io.Writer
	TimeScale int

	SPS []byte
	PPS []byte

	tswPAT  *TSWriter
	tswPMT  *TSWriter
	tswH264 *TSWriter

	PTS int64
	PCR int64

	prepared bool
	writeSPS bool

	pesBuf *bytes.Buffer
	patBuf []byte
	pmtBuf []byte
}

func (self *SimpleH264Writer) prepare() (err error) {
	pat := PAT{
		Entries: []PATEntry{
			{ProgramNumber: 1, ProgramMapPID: 0x1000},
		},
	}
	bw := &bytes.Buffer{}
	if err = WritePAT(bw, pat); err != nil {
		return
	}
	self.patBuf = bw.Bytes()

	pmt := PMT{
		PCRPID: 0x100,
		ElementaryStreamInfos: []ElementaryStreamInfo{
			{StreamType: ElementaryStreamTypeH264, ElementaryPID: 0x100},
		},
	}
	bw = &bytes.Buffer{}
	if err = WritePMT(bw, pmt); err != nil {
		return
	}
	self.pmtBuf = bw.Bytes()

	self.tswPMT = &TSWriter{
		PID: 0x1000,
		DiscontinuityIndicator: true,
	}
	self.tswPAT = &TSWriter{
		PID: 0,
		DiscontinuityIndicator: true,
	}
	self.tswH264 = &TSWriter{
		PID: 0x100,
		DiscontinuityIndicator: true,
	}

	self.tswH264.EnableVecWriter()

	if self.PTS == 0 {
		// 1s
		self.PTS = int64(self.TimeScale)
	}
	if self.PCR == 0 {
		// 1s
		self.PCR = int64(self.TimeScale)
	}

	self.pesBuf = &bytes.Buffer{}

	self.prepared = true
	return
}

func (self *SimpleH264Writer) WriteHeader() (err error) {
	if !self.prepared {
		if err = self.prepare(); err != nil {
			return
		}
	}
	self.tswPAT.W = self.W
	if err = self.tswPAT.Write(self.patBuf); err != nil {
		return
	}
	self.tswPMT.W = self.W
	if err = self.tswPMT.Write(self.pmtBuf); err != nil {
		return
	}
	self.writeSPS = true
	return
}

func (self *SimpleH264Writer) WriteNALU(sync bool, duration int, nalu []byte) (err error) {
	nalus := [][]byte{}

	if !self.prepared {
		if err = self.WriteHeader(); err != nil {
			return
		}
	}

	if self.writeSPS {
		self.writeSPS = false
		nalus = append(nalus, self.SPS)
		nalus = append(nalus, self.PPS)
	}
	nalus = append(nalus, nalu)

	pes := PESHeader{
		StreamId: StreamIdH264,
		PTS:      uint64(self.PTS) * PTS_HZ / uint64(self.TimeScale),
	}
	if err = WritePESHeader(self.pesBuf, pes, 0); err != nil {
		return
	}

	data := &iovec{}
	data.Append(self.pesBuf.Bytes())
	for i, nalu := range nalus {
		var startCode []byte
		if i == 0 {
			startCode = []byte{0, 0, 0, 1, 0x9, 0xf0, 0, 0, 0, 1} // AUD
		} else {
			startCode = []byte{0, 0, 1}
		}
		data.Append(startCode)
		data.Append(nalu)
	}

	self.tswH264.RandomAccessIndicator = sync
	self.tswH264.PCR = uint64(self.PCR) * PCR_HZ / uint64(self.TimeScale)

	self.tswH264.W = self.W
	if err = self.tswH264.WriteIovec(data); err != nil {
		return
	}

	self.PTS += int64(duration)
	self.PCR += int64(duration)
	self.pesBuf.Reset()

	return
}
