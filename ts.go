
package ts

import (
	"fmt"
)

const debug = true

const (
	ElementaryStreamTypeH264 = 0x1B
	ElementaryStreamTypeAdtsAAC = 0x0F
)

type TSHeader struct {
	PID uint
	PCR uint64
	OPCR uint64
	ContinuityCounter uint
	PayloadUnitStart bool
}

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
	SecNum uint
	LastSecNum uint
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
	return (((v>>33)&0x7)<<30) | (((v>>17)&0x7fff)<<15) | ((v>>1)&0x7fff)
}

func UIntToPCR(v uint64) uint64 {
	// base(33)+resverd(6)+ext(9)
	base := v>>15
	ext := v&0x1ff
	return base*300+ext
}

type FieldsDumper struct {
	Fields []struct {
		Length int
		Desc string
	}
	Val uint
	Length uint
}

func (self FieldsDumper) String() (res string) {
	pos := uint(self.Length)
	for _, field := range self.Fields {
		pos -= uint(field.Length)
		val := (self.Val>>pos)&(1<<uint(field.Length)-1)
		if val != 0 {
			res += fmt.Sprintf("%s=%x ", field.Desc, val)
		}
	}
	return
}

