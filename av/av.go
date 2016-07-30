package av

import (
	"fmt"
	"time"
)

// Audio sample format
type SampleFormat uint8

const (
	U8 = SampleFormat(iota + 1) // 8-bit unsigned integer
	S16 // signed 16-bit integer
	S32 // signed 32-bit integer
	FLT // 32-bit float
	DBL // 64-bit float
	U8P // 8-bit unsigned integer in planar
	S16P // signed 16-bit integer in planar
	S32P // signed 32-bit integer in planar
	FLTP // 32-bit float in planar
	DBLP // 64-bit float in planar
	U32 // unsigned 32-bit integer
)

func (self SampleFormat) BytesPerSample() int {
	switch self {
	case U8, U8P:
		return 1
	case S16, S16P:
		return 2
	case FLT, FLTP, S32, S32P, U32:
		return 4
	case DBL, DBLP:
		return 8
	default:
		return 0
	}
}

func (self SampleFormat) String() string {
	switch self {
	case U8:
		return "U8"
	case S16:
		return "S16"
	case S32:
		return "S32"
	case FLT:
		return "FLT"
	case DBL:
		return "DBL"
	case U8P:
		return "U8P"
	case S16P:
		return "S16P"
	case FLTP:
		return "FLTP"
	case DBLP:
		return "DBLP"
	case U32:
		return "U32"
	default:
		return "?"
	}
}

// checkout if this sample format is in planar
func (self SampleFormat) IsPlanar() bool {
	switch self {
	case S16P, S32P, FLTP, DBLP:
		return true
	default:
		return false
	}
}

// audio channel layout
type ChannelLayout uint16

func (self ChannelLayout) String() string {
	return fmt.Sprintf("%dch", self.Count())
}

const (
	CH_FRONT_CENTER = ChannelLayout(1 << iota)
	CH_FRONT_LEFT
	CH_FRONT_RIGHT
	CH_BACK_CENTER
	CH_BACK_LEFT
	CH_BACK_RIGHT
	CH_SIDE_LEFT
	CH_SIDE_RIGHT
	CH_LOW_FREQ
	CH_NR

	CH_MONO     = ChannelLayout(CH_FRONT_CENTER)
	CH_STEREO   = ChannelLayout(CH_FRONT_LEFT | CH_FRONT_RIGHT)
	CH_2_1      = ChannelLayout(CH_STEREO | CH_BACK_CENTER)
	CH_2POINT1  = ChannelLayout(CH_STEREO | CH_LOW_FREQ)
	CH_SURROUND = ChannelLayout(CH_STEREO | CH_FRONT_CENTER)
	CH_3POINT1  = ChannelLayout(CH_SURROUND | CH_LOW_FREQ)
	// TODO: add all channel_layout in ffmpeg
)

func (self ChannelLayout) Count() (n int) {
	for self != 0 {
		n++
		self = (self - 1) & self
	}
	return
}

// Video/Audio codec type. can be H264/AAC/SPEEX/...
type CodecType uint32

var (
	H264 = MakeVideoCodecType(avCodecTypeMagic + 1)
	AAC       = MakeAudioCodecType(avCodecTypeMagic + 1)
	PCM_MULAW = MakeAudioCodecType(avCodecTypeMagic + 2)
	PCM_ALAW  = MakeAudioCodecType(avCodecTypeMagic + 3)
	SPEEX = MakeAudioCodecType(avCodecTypeMagic + 4)
	NELLYMOSER = MakeAudioCodecType(avCodecTypeMagic + 5)
)

const codecTypeAudioBit = 0x1
const codecTypeOtherBits = 1

func (self CodecType) String() string {
	switch self {
	case H264:
		return "H264"
	case AAC:
		return "AAC"
	case PCM_MULAW:
		return "PCM_MULAW"
	case PCM_ALAW:
		return "PCM_ALAW"
	case SPEEX:
		return "SPEEX"
	case NELLYMOSER:
		return "NELLYMOSER"
	}
	return ""
}

// CodecType is audio
func (self CodecType) IsAudio() bool {
	return self&codecTypeAudioBit != 0
}

// CodecType is video
func (self CodecType) IsVideo() bool {
	return self&codecTypeAudioBit == 0
}

// make a new audio codec type
func MakeAudioCodecType(base uint32) (c CodecType) {
	c = CodecType(base)<<codecTypeOtherBits | CodecType(codecTypeAudioBit)
	return
}

// make a new video codec type
func MakeVideoCodecType(base uint32) (c CodecType) {
	c = CodecType(base) << codecTypeOtherBits
	return
}

const avCodecTypeMagic = 233333

// CodecData is some important bytes for initializing audio/video decoder.
// 
// video width/height and audio sample rate, channel layout can get from CodecData.
// 
// CodecData can convert to VideoCodecData or AudioCodecData using:
//
//     ```codecdata.(AudioCodecData) or codecdata.(VideoCodecData)```
// 
// for H264, CodecData is AVCDecoderConfigure bytes, includes SPS/PPS
type CodecData interface {
	Type() CodecType // Video/Audio codec type
}

type VideoCodecData interface {
	CodecData
	Width() int // Video height
	Height() int // Video width
}

type AudioCodecData interface {
	CodecData
	SampleFormat() SampleFormat // Audio sample format
	SampleRate() int // Audio sample rate
	ChannelLayout() ChannelLayout // Audio channel layout

	// get audio packet duration
	PacketDuration([]byte) (time.Duration, error)
}

type PacketWriter interface {
	WritePacket(Packet) error
}

type PacketReader interface {
	ReadPacket() (Packet,error)
}

// Muxer describes the steps of writing compressed audio/video packets into container formats like MP4/FLV/MPEG-TS.
//
// 1. WriteHeader([]CodecData) write the file header, each stream 
//
// 2. WritePacket(Packet) write the audio/video packets
//
// 3. WriteTrailer() end writing, now it's a complete file.
//
// WriteHeader/WriteTrailer can be called only once.
// 
// every formsts(format/flv format/mp4 ...), rtmp.Conn, and transcode.Muxer implements Muxer interface.
type Muxer interface {
	PacketWriter
	WriteHeader([]CodecData) error
	WriteTrailer() error
}

// Muxer with Close() method
type MuxCloser interface {
	Muxer
	Close() error
}

// Demuxer can demux compressed audio/video packets from container formats like MP4/FLV/MPEG-TS.
// 
// Streams() ([]CodecData, error) reads the file header, contains video/audio meta infomations
//
// ReadPacket() (Packet, error) read compressed audio/video packets
type Demuxer interface {
	PacketReader
	Streams() ([]CodecData, error)
}

// Demuxer with Close() method
type DemuxCloser interface {
	Demuxer
	Close() error
}

// Packet stores compressed audio/video data
type Packet struct {
	IsKeyFrame      bool // video packet is key frame
	Idx             int8 // stream index in container format
	CompositionTime time.Duration // packet presentation time minus decode time for H264 B-Frame
	Time time.Duration // packet decode time
	Data            []byte // packet data
}

// Raw audio frame
type AudioFrame struct {
	SampleFormat  SampleFormat // audio sample format, e.g: S16,FLTP,...
	ChannelLayout ChannelLayout // audio channel layout, e.g: CH_MONO,CH_STEREO,...
	SampleCount   int // sample count in this frame
	SampleRate    int // sample rate
	Data          [][]byte // data array for planar format len(Data) > 1
}

// audio frame duration
func (self AudioFrame) Duration() time.Duration {
	return time.Second * time.Duration(self.SampleCount) / time.Duration(self.SampleRate)
}

// check this audio frame has same format as other audio frame
func (self AudioFrame) HasSameFormat(other AudioFrame) bool {
	if self.SampleRate != other.SampleRate {
		return false
	}
	if self.ChannelLayout != other.ChannelLayout {
		return false
	}
	if self.SampleFormat != other.SampleFormat {
		return false
	}
	return true
}

// split sample audio sample from this frame
func (self AudioFrame) Slice(start int, end int) (out AudioFrame) {
	out = self
	out.Data = append([][]byte(nil), out.Data...)
	out.SampleCount = end - start
	size := self.SampleFormat.BytesPerSample()
	for i := range out.Data {
		out.Data[i] = out.Data[i][start*size : end*size]
	}
	return
}

// concat two audio frames
func (self AudioFrame) Concat(in AudioFrame) (out AudioFrame) {
	out = self
	out.Data = append([][]byte(nil), out.Data...)
	out.SampleCount += in.SampleCount
	for i := range out.Data {
		out.Data[i] = append(out.Data[i], in.Data[i]...)
	}
	return
}

// AudioEncoder can encode raw audio frame into compressed audio packets
//
// now cgo/ffmpeg inplements AudioEncoder, using ffmpeg.NewAudioEncoder to create it
type AudioEncoder interface {
	CodecData() (AudioCodecData, error) // encoder's codec data can put into container
	Encode(AudioFrame) ([][]byte, error) // encode raw audio frame into compressed pakcet(s)
	//Flush() ([]Packet, error)
	Close() // close encoder, free cgo contexts
	SetSampleRate(int) (error) // set encoder sample rate
	SetChannelLayout(ChannelLayout) (error) // set encoder channel layout
	SetSampleFormat(SampleFormat) (error) // set encoder sample format
	SetBitrate(int) (error) // set encoder bitrate
	SetOption(string,interface{}) (error) // encoder setopt, in ffmpeg is av_opt_set_dict()
	GetOption(string,interface{}) (error) // encoder getopt
}

// AudioDecoder can decode compressed audio packets into raw audio frame
//
// use ffmpeg.NewAudioDecoder to create it
type AudioDecoder interface {
	Decode([]byte) (bool, AudioFrame, error) // decode one compressed audio packet
	//Flush() (AudioFrame, error)
	Close() // close decode, free cgo contexts
}

// AudioResampler can convert raw audio frames in different sample rate/format/channel layout
type AudioResampler interface {
	Resample(AudioFrame) (AudioFrame, error) // convert raw audio frames
}

