package codec

import (
	"github.com/nareix/av"
)

type AudioCodecData struct {
	CodecType int
	CodecSampleRate int
	CodecChannelLayout av.ChannelLayout
	CodecSampleFormat av.SampleFormat
}

func (self AudioCodecData) Type() int {
	return self.CodecType
}

func (self AudioCodecData) IsAudio() bool {
	return true
}

func (self AudioCodecData) IsVideo() bool {
	return false
}

func (self AudioCodecData) SampleRate() int {
	return self.CodecSampleRate
}

func (self AudioCodecData) ChannelLayout() av.ChannelLayout {
	return self.CodecChannelLayout
}

func (self AudioCodecData) SampleFormat() av.SampleFormat {
	return self.CodecSampleFormat
}

func NewPCMMulawCodecData() av.AudioCodecData {
	return AudioCodecData{
		CodecType: av.PCM_MULAW,
		CodecSampleFormat: av.S16,
		CodecChannelLayout: av.CH_MONO,
		CodecSampleRate: 8000,
	}
}

func NewPCMAlawCodecData() av.AudioCodecData {
	return AudioCodecData{
		CodecType: av.PCM_ALAW,
		CodecSampleFormat: av.S16,
		CodecChannelLayout: av.CH_MONO,
		CodecSampleRate: 8000,
	}
}

