package codec

import (
	"github.com/nareix/av"
	"time"
)

type PCMUCodecData struct {
	typ av.CodecType
}

func (self PCMUCodecData) Type() av.CodecType {
	return self.typ
}

func (self PCMUCodecData) SampleRate() int {
	return 8000
}

func (self PCMUCodecData) ChannelLayout() av.ChannelLayout {
	return av.CH_MONO
}

func (self PCMUCodecData) SampleFormat() av.SampleFormat {
	return av.S16
}

func (self PCMUCodecData) PacketDuration(data []byte) (time.Duration, error) {
	return time.Duration(len(data)) * time.Second / time.Duration(8000), nil
}

func NewPCMMulawCodecData() av.AudioCodecData {
	return PCMUCodecData{
		typ: av.PCM_MULAW,
	}
}

func NewPCMAlawCodecData() av.AudioCodecData {
	return PCMUCodecData{
		typ: av.PCM_ALAW,
	}
}

