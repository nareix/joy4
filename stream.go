package rtsp

import (
	"github.com/nareix/av"
)

type Stream struct {
	av.StreamCommon
	typestr string
	control string
}

func (self Stream) IsAudio() bool {
	return self.typestr == "audio"
}

func (self Stream) IsVideo() bool {
	return self.typestr == "video"
}

