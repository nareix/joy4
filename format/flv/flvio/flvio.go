package flvio

import (
	"io"
	"time"
	"fmt"
	"github.com/nareix/pio"
)

func TsToTime(ts int32) time.Duration {
	return time.Millisecond*time.Duration(ts)
}

func TimeToTs(tm time.Duration) int32 {
	return int32(tm / time.Millisecond)
}

const (
	TAG_AUDIO      = 8
	TAG_VIDEO      = 9
	TAG_SCRIPTDATA = 18
)

type Tag interface {
	Type() uint8
	Len() int
	Marshal(*pio.Writer) error
	Unmarshal(*pio.Reader) error
}

type Scriptdata struct {
	Data []byte
}

func (self Scriptdata) Type() uint8 {
	return TAG_SCRIPTDATA
}

func (self Scriptdata) Marshal(w *pio.Writer) (err error) {
	if _, err = w.Write(self.Data); err != nil {
		return
	}
	return
}

func (self Scriptdata) Len() int {
	return len(self.Data)
}

func (self *Scriptdata) Unmarshal(r *pio.Reader) (err error) {
	if self.Data, err = r.ReadBytes(int(r.N)); err != nil {
		return
	}
	return
}

const (
	SOUND_MP3 = 2
	SOUND_NELLYMOSER_16KHZ_MONO = 4
	SOUND_NELLYMOSER_8KHZ_MONO = 5
	SOUND_NELLYMOSER = 6
	SOUND_ALAW = 7
	SOUND_MULAW = 8
	SOUND_AAC = 10
	SOUND_SPEEX = 11

	SOUND_5_5Khz = 0
	SOUND_11Khz  = 1
	SOUND_22Khz  = 2
	SOUND_44Khz  = 3

	SOUND_8BIT  = 0
	SOUND_16BIT = 1

	SOUND_MONO   = 0
	SOUND_STEREO = 1

	AAC_SEQHDR = 0
	AAC_RAW    = 1
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
		7 = G.711 A-law logarithmic PCM
		8 = G.711 mu-law logarithmic PCM
		9 = reserved
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
	if self.SoundFormat == SOUND_AAC {
		return 2 + len(self.Data)
	} else {
		return 1 + len(self.Data)
	}
}

func (self Audiodata) Marshal(w *pio.Writer) (err error) {
	var flags uint8
	flags |= self.SoundFormat << 4
	flags |= self.SoundRate << 2
	flags |= self.SoundSize << 1
	flags |= self.SoundType
	if err = w.WriteU8(flags); err != nil {
		return
	}

	switch self.SoundFormat {
	case SOUND_AAC:
		if err = w.WriteU8(self.AACPacketType); err != nil {
			return
		}
		if _, err = w.Write(self.Data); err != nil {
			return
		}

	default:
		if _, err = w.Write(self.Data); err != nil {
			return
		}
	}
	return
}

func (self *Audiodata) Unmarshal(r *pio.Reader) (err error) {
	var flags uint8
	if flags, err = r.ReadU8(); err != nil {
		return
	}
	self.SoundFormat = flags >> 4
	self.SoundRate = (flags >> 2) & 0x3
	self.SoundSize = (flags >> 1) & 0x1
	self.SoundType = flags & 0x1

	switch self.SoundFormat {
	case SOUND_AAC:
		if self.AACPacketType, err = r.ReadU8(); err != nil {
			return
		}
		if self.Data, err = r.ReadBytes(int(r.N)); err != nil {
			return
		}

	default:
		if self.Data, err = r.ReadBytes(int(r.N)); err != nil {
			return
		}
	}

	return
}

const (
	AVC_SEQHDR = 0
	AVC_NALU   = 1
	AVC_EOS    = 2

	FRAME_KEY   = 1
	FRAME_INTER = 2

	VIDEO_H264 = 7
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

func (self *Videodata) Unmarshal(r *pio.Reader) (err error) {
	var flags uint8
	if flags, err = r.ReadU8(); err != nil {
		return
	}
	self.FrameType = flags >> 4
	self.CodecID = flags & 0xf
	if self.AVCPacketType, err = r.ReadU8(); err != nil {
		return
	}
	if self.CompositionTime, err = r.ReadI24BE(); err != nil {
		return
	}
	switch self.AVCPacketType {
	case AVC_SEQHDR, AVC_NALU:
		if self.Data, err = r.ReadBytes(int(r.N)); err != nil {
			return
		}
	}
	return
}

func (self Videodata) Marshal(w *pio.Writer) (err error) {
	flags := self.FrameType<<4 | self.CodecID
	if err = w.WriteU8(flags); err != nil {
		return
	}
	if err = w.WriteU8(self.AVCPacketType); err != nil {
		return
	}
	if err = w.WriteI24BE(self.CompositionTime); err != nil {
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

const (
	// TypeFlagsReserved UB[5]
	// TypeFlagsAudio    UB[1] Audio tags are present
	// TypeFlagsReserved UB[1] Must be 0
	// TypeFlagsVideo    UB[1] Video tags are present
	FILE_HAS_AUDIO = 0x4
	FILE_HAS_VIDEO = 0x1
)

func ReadFileHeader(r *pio.Reader) (flags uint8, err error) {
	var cc3 uint32
	if cc3, err = r.ReadU24BE(); err != nil {
		return
	}
	if cc3 != 0x464c56 { // 'FLV'
		err = fmt.Errorf("flvio: file header cc3 invalid")
		return
	}

	// version
	if _, err = r.ReadI8(); err != nil {
		return
	}

	if flags, err = r.ReadU8(); err != nil {
		return
	}

	var dataoffset uint32
	if dataoffset, err = r.ReadU32BE(); err != nil {
		return
	}
	dataoffset -= 9

	// skip header and first `tagsize`
	if _, err = r.Discard(int(dataoffset + 4)); err != nil {
		return
	}

	return
}

func ReadTag(r *pio.Reader) (tag Tag, timestamp int32, err error) {
	var tagtype uint8
	if tagtype, err = r.ReadU8(); err != nil {
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
		err = fmt.Errorf("flvio: ReadTag tagtype=%d invalid", tagtype)
		return
	}

	var datasize uint32
	if datasize, err = r.ReadU24BE(); err != nil {
		return
	}

	var tslo uint32
	var tshi uint8
	if tslo, err = r.ReadU24BE(); err != nil {
		return
	}
	if tshi, err = r.ReadU8(); err != nil {
		return
	}
	timestamp = int32(tslo|uint32(tshi)<<24)

	if _, err = r.ReadI24BE(); err != nil {
		return
	}

	b := make([]byte, datasize)
	if _, err = io.ReadFull(r, b); err != nil {
		return
	}
	br := pio.NewReaderBytes(b)
	br.LimitOn(int64(datasize))
	if err = tag.Unmarshal(br); err != nil {
		return
	}
	br.LimitOff()

	if _, err = r.ReadI32BE(); err != nil {
		return
	}

	return
}

func WriteTag(w *pio.Writer, tag Tag, timestamp int32) (err error) {
	if err = w.WriteU8(tag.Type()); err != nil {
		return
	}
	datasize := tag.Len()
	if err = w.WriteU24BE(uint32(datasize)); err != nil {
		return
	}
	if err = w.WriteU24BE(uint32(timestamp & 0xffffff)); err != nil {
		return
	}
	if err = w.WriteU8(uint8(timestamp >> 24)); err != nil {
		return
	}
	if err = w.WriteI24BE(0); err != nil {
		return
	}
	if err = tag.Marshal(w); err != nil {
		return
	}
	if err = w.WriteU32BE(uint32(datasize) + 11); err != nil {
		return
	}
	return
}

func WriteFileHeader(w *pio.Writer, flags uint8) (err error) {
	// 'FLV', version 1
	if err = w.WriteI32BE(0x464c5601); err != nil {
		return
	}

	if err = w.WriteU8(flags); err != nil {
		return
	}

	// DataOffset: UI32 Offset in bytes from start of file to start of body (that is, size of header)
	// The DataOffset field usually has a value of 9 for FLV version 1.
	if err = w.WriteU32BE(9); err != nil {
		return
	}

	// PreviousTagSize0: UI32 Always 0
	if err = w.WriteU32BE(0); err != nil {
		return
	}

	return
}


