package flv

import (
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/bits"
	"io"
)

const (
	TAG_AUDIO      = 8
	TAG_VIDEO      = 9
	TAG_SCRIPTDATA = 18
)

type Tag interface {
	Type() uint8
	Len() int
	Marshal(*Writer) error
	Unmarshal(*Reader) error
}

type Scriptdata struct {
	Data []byte
}

func (self Scriptdata) Type() uint8 {
	return TAG_SCRIPTDATA
}

func (self Scriptdata) Marshal(w *Writer) (err error) {
	if _, err = w.Write(self.Data); err != nil {
		return
	}
	return
}

func (self Scriptdata) Len() int {
	return len(self.Data)
}

func (self *Scriptdata) Unmarshal(r *Reader) (err error) {
	self.Data = make([]byte, r.N())
	if _, err = io.ReadFull(r, self.Data); err != nil {
		return
	}
	return
}

const (
	SOUND_AAC = 10

	SOUND_5_5Khz = 0
	SOUND_11Khz  = 1
	SOUND_22Khz  = 2
	SOUND_44Khz  = 3

	SOUND_8BIT  = 0
	SOUND_16BIT = 1

	SOUND_MONO   = 0
	SOUND_STEREO = 1

	AAC_SEQHDR = 0
	AAC_RAW    = 0
)

type Audiodata struct {
	/*
		SoundFormat: UB[4]
		0 = Linear PCM, platform endian
		1 = ADPCM
		2 = MP3
		3 = Linear PCM, little endian
		4 = Nellymoser 16-kHz mono
		5 = Nellymoser 8-kHz mono
		6 = Nellymoser
		7 = G.711 A-law logarithmic PCM 8 = G.711 mu-law logarithmic PCM 9 = reserved
		10 = AAC
		11 = Speex
		14 = MP3 8-Khz
		15 = Device-specific sound
		Formats 7, 8, 14, and 15 are reserved for internal use
		AAC is supported in Flash Player 9,0,115,0 and higher.
		Speex is supported in Flash Player 10 and higher.
	*/
	SoundFormat uint8

	/*
		SoundRate: UB[2]
		Sampling rate
		0 = 5.5-kHz For AAC: always 3
		1 = 11-kHz
		2 = 22-kHz
		3 = 44-kHz
	*/
	SoundRate uint8

	/*
		SoundSize: UB[1]
		0 = snd8Bit
		1 = snd16Bit
		Size of each sample.
		This parameter only pertains to uncompressed formats.
		Compressed formats always decode to 16 bits internally
	*/
	SoundSize uint8

	/*
		SoundType: UB[1]
		0 = sndMono
		1 = sndStereo
		Mono or stereo sound For Nellymoser: always 0
		For AAC: always 1
	*/
	SoundType uint8

	/*
		0: AAC sequence header
		1: AAC raw
	*/
	AACPacketType uint8

	Data []byte
}

func (self Audiodata) Type() uint8 {
	return TAG_AUDIO
}

func (self Audiodata) Len() int {
	return 2 + len(self.Data)
}

func (self Audiodata) Marshal(w *Writer) (err error) {
	var flags uint8
	flags |= self.SoundFormat << 4
	flags |= self.SoundRate << 2
	flags |= self.SoundSize << 1
	flags |= self.SoundType
	if err = w.WriteUInt8(flags); err != nil {
		return
	}
	if self.SoundFormat == SOUND_AAC {
		if err = w.WriteUInt8(self.AACPacketType); err != nil {
			return
		}
		if _, err = w.Write(self.Data); err != nil {
			return
		}
	} else {
		err = fmt.Errorf("flv: Audiodata.Marshal: unsupported SoundFormat=%d", self.SoundFormat)
		return
	}
	return
}

func (self *Audiodata) Unmarshal(r *Reader) (err error) {
	var flags uint8
	if flags, err = r.ReadUInt8(); err != nil {
		return
	}
	self.SoundFormat = flags >> 4
	self.SoundRate = (flags >> 2) & 0x3
	self.SoundSize = (flags >> 1) & 0x1
	self.SoundType = flags & 0x1
	if self.SoundFormat == SOUND_AAC {
		if self.AACPacketType, err = r.ReadUInt8(); err != nil {
			return
		}
		self.Data = make([]byte, r.N())
		if _, err = io.ReadFull(r, self.Data); err != nil {
			return
		}
	} else {
		err = fmt.Errorf("flv: Audiodata.Unmarshal: unsupported SoundFormat=%d", self.SoundFormat)
		return
	}
	return
}

const (
	AVC_SEQHDR = 0
	AVC_NALU   = 1
	AVC_EOS    = 2

	FRAME_KEY   = 1
	FRAME_INTER = 2

	CODEC_AAC = 7
)

type Videodata struct {
	/*
		1: keyframe (for AVC, a seekable frame)
		2: inter frame (for AVC, a non- seekable frame)
		3: disposable inter frame (H.263 only)
		4: generated keyframe (reserved for server use only)
		5: video info/command frame
	*/
	FrameType uint8

	/*
		1: JPEG (currently unused)
		2: Sorenson H.263
		3: Screen video
		4: On2 VP6
		5: On2 VP6 with alpha channel
		6: Screen video version 2
		7: AVC
	*/
	CodecID uint8

	/*
		0: AVC sequence header
		1: AVC NALU
		2: AVC end of sequence (lower level NALU sequence ender is not required or supported)
	*/
	AVCPacketType uint8

	Data            []byte
	CompositionTime int32
}

func (self Videodata) Type() uint8 {
	return TAG_VIDEO
}

func (self Videodata) Len() int {
	return 5 + len(self.Data)
}

func (self *Videodata) Unmarshal(r *Reader) (err error) {
	var flags uint8
	if flags, err = r.ReadUInt8(); err != nil {
		return
	}
	self.FrameType = flags >> 4
	self.CodecID = flags & 0xff
	if self.AVCPacketType, err = r.ReadUInt8(); err != nil {
		return
	}
	if self.CompositionTime, err = r.ReadInt24BE(); err != nil {
		return
	}
	switch self.AVCPacketType {
	case AVC_SEQHDR, AVC_NALU:
		self.Data = make([]byte, r.N())
		if _, err = io.ReadFull(r, self.Data); err != nil {
			return
		}
	}
	return
}

func (self Videodata) Marshal(w *Writer) (err error) {
	flags := self.FrameType<<4 | self.CodecID
	if err = w.WriteUInt8(flags); err != nil {
		return
	}
	if err = w.WriteUInt8(self.AVCPacketType); err != nil {
		return
	}
	if err = w.WriteInt24BE(self.CompositionTime); err != nil {
		return
	}
	switch self.AVCPacketType {
	case AVC_SEQHDR, AVC_NALU:
		if _, err = w.Write(self.Data); err != nil {
			return
		}
	}
	return
}

type Reader struct {
	bits.IntReader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{IntReader: bits.NewIntReader(r)}
}

func (self *Reader) NewLimitedReader(n int) *Reader {
	nr := *self
	nr.IntReader.R = &io.LimitedReader{R: nr.IntReader.R, N: int64(n)}
	return &nr
}

func (self Reader) N() int {
	return int(self.IntReader.R.(*io.LimitedReader).N)
}

func (self *Reader) ReadHeader() (err error) {
	var cc3 uint32
	if cc3, err = self.ReadUInt24BE(); err != nil {
		return
	}
	if cc3 != 0x464c56 { // 'FLV'
		err = fmt.Errorf("flv: file header cc3 invalid")
		return
	}

	// version
	if _, err = self.ReadInt8(); err != nil {
		return
	}

	// flags
	if _, err = self.ReadInt8(); err != nil {
		return
	}

	var dataoffset uint
	if dataoffset, err = self.ReadUInt32BE(); err != nil {
		return
	}
	dataoffset -= 9

	// skip header and first `tagsize`
	if err = self.Skip(int(dataoffset + 4)); err != nil {
		return
	}

	return
}

func (self *Reader) ReadTag() (tag Tag, timestamp int32, err error) {
	var tagtype uint8
	if tagtype, err = self.ReadUInt8(); err != nil {
		return
	}

	switch tagtype {
	case TAG_AUDIO:
		tag = &Audiodata{}

	case TAG_VIDEO:
		tag = &Videodata{}

	case TAG_SCRIPTDATA:
		tag = &Scriptdata{}

	default:
		err = fmt.Errorf("flv: ReadTag tagtype=%d invalid", tagtype)
		return
	}

	var datasize uint32
	if datasize, err = self.ReadUInt24BE(); err != nil {
		return
	}

	var tslo uint32
	var tshi uint8
	if tslo, err = self.ReadUInt24BE(); err != nil {
		return
	}
	if tshi, err = self.ReadUInt8(); err != nil {
		return
	}
	timestamp = int32(tslo|uint32(tshi)<<24)

	if _, err = self.ReadInt24BE(); err != nil {
		return
	}

	if err = tag.Unmarshal(self.NewLimitedReader(int(datasize))); err != nil {
		return
	}

	if _, err = self.ReadInt32BE(); err != nil {
		return
	}

	return
}

type Writer struct {
	bits.IntWriter
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{IntWriter: bits.NewIntWriter(w)}
}

func (self *Writer) WriteTag(tag Tag, timestamp int32) (err error) {
	if err = self.WriteUInt8(tag.Type()); err != nil {
		return
	}
	datasize := tag.Len()
	if err = self.WriteUInt24BE(uint32(datasize)); err != nil {
		return
	}
	if err = self.WriteUInt24BE(uint32(timestamp & 0xffffff)); err != nil {
		return
	}
	if err = self.WriteUInt8(uint8(timestamp >> 24)); err != nil {
		return
	}
	if err = self.WriteInt24BE(0); err != nil {
		return
	}
	if err = tag.Marshal(self); err != nil {
		return
	}
	if err = self.WriteUInt32BE(uint32(datasize) + 11); err != nil {
		return
	}
	return
}

func (self *Writer) WriteHeader(hasVideo bool, hasAudio bool) (err error) {
	// 'FLV', version 1
	if err = self.WriteInt32BE(0x464c5601); err != nil {
		return
	}

	// TypeFlagsReserved UB[5]
	// TypeFlagsAudio    UB[1] Audio tags are present
	// TypeFlagsReserved UB[1] Must be 0
	// TypeFlagsVideo    UB[1] Video tags are present
	var flags uint8
	if hasAudio {
		flags |= 1 << 2
	}
	if hasVideo {
		flags |= 1
	}
	if err = self.WriteUInt8(flags); err != nil {
		return
	}

	// DataOffset: UI32 Offset in bytes from start of file to start of body (that is, size of header)
	// The DataOffset field usually has a value of 9 for FLV version 1.
	if err = self.WriteUInt32BE(9); err != nil {
		return
	}

	// PreviousTagSize0: UI32 Always 0
	if err = self.WriteUInt32BE(0); err != nil {
		return
	}

	return
}

type Muxer struct {
	fw *Writer
}

func NewMuxer(w io.Writer) *Muxer {
	self := &Muxer{}
	self.fw = NewWriter(w)
	return self
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	hasVideo := false
	hasAudio := false
	for _, stream := range streams {
		if stream.IsVideo() {
			hasVideo = true
		} else if stream.IsAudio() {
			hasAudio = true
		}
	}

	if err = self.fw.WriteHeader(hasVideo, hasAudio); err != nil {
		return
	}

	return
}

