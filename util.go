package av

import (
	"fmt"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec/h264parser"
)

type CodecType int

const (
	H264 = CodecType(1)
	AAC  = CodecType(2)
)

func (self CodecType) String() string {
	switch self {
	case H264:
		return "H264"
	case AAC:
		return "AAC"
	}
	return "?"
}

func (self CodecType) IsAudio() bool {
	return self == AAC
}

func (self CodecType) IsVideo() bool {
	return self == H264
}

type Stream interface {
	IsVideo() bool
	IsAudio() bool
	Type() CodecType
	SetType(CodecType)
	CodecData() []byte
	SetCodecData([]byte) error
	SampleRate() int
	ChannelCount() int
	Width() int
	Height() int
	String() string
	FillParamsByStream(Stream) error
}

type StreamCommon struct {
	codecType     CodecType
	codecData     []byte
	H264CodecInfo h264parser.CodecInfo
	AACCodecInfo  aacparser.CodecInfo
}

type Muxer interface {
	NewStream() Stream
	WriteHeader() error
	WriteTrailer() error
	WritePacket(int, Packet) error
	SetTime(float64)
}

type Demuxer interface {
	ReadHeader() error
	ReadPacket() (int, Packet, error)
	Streams() []Stream
	Time() float64
}

type SeekableDemuxer interface {
	Demuxer
	SeekToTime(float64) error
}

type Packet struct {
	IsKeyFrame      bool
	Data            []byte
	Duration        float64
	CompositionTime float64
}

func (self *StreamCommon) Type() CodecType {
	return self.codecType
}

func (self *StreamCommon) SetType(CodecType CodecType) {
	self.codecType = CodecType
}

func (self *StreamCommon) String() string {
	str := self.codecType.String()
	if self.IsAudio() {
		str += fmt.Sprintf(" %dHz %dch", self.SampleRate(), self.ChannelCount())
	} else if self.IsVideo() {
		str += fmt.Sprintf(" %dx%d", self.Width(), self.Height())
	}
	return str
}

func (self *StreamCommon) SetCodecData(data []byte) (err error) {
	if self.codecType == H264 {
		if self.H264CodecInfo, err = h264parser.ParseCodecData(data); err != nil {
			return
		}
	} else if self.codecType == AAC {
		if self.AACCodecInfo, err = aacparser.ParseCodecData(data); err != nil {
			return
		}
	} else {
		err = fmt.Errorf("unknown codec type=%d", self.codecType)
	}
	self.codecData = data
	return
}

func (self *StreamCommon) CodecData() (data []byte) {
	return self.codecData
}

func (self *StreamCommon) ChannelCount() int {
	return self.AACCodecInfo.ChannelCount
}

func (self *StreamCommon) SampleRate() int {
	return self.AACCodecInfo.SampleRate
}

func (self *StreamCommon) Width() int {
	return int(self.H264CodecInfo.SPSInfo.Width)
}

func (self *StreamCommon) Height() int {
	return int(self.H264CodecInfo.SPSInfo.Height)
}

func (self *StreamCommon) IsVideo() bool {
	if self.codecType == H264 {
		return true
	}
	return false
}

func (self *StreamCommon) IsAudio() bool {
	if self.codecType == AAC {
		return true
	}
	return false
}

func (self *StreamCommon) FillParamsByStream(other Stream) (err error) {
	self.codecType = other.Type()
	if err = self.SetCodecData(other.CodecData()); err != nil {
		return
	}
	return
}

