package rtsp

import (
	"github.com/nareix/av"
	"github.com/nareix/rtsp/sdp"
)

type Stream struct {
	av.CodecData
	Sdp sdp.Info

	// h264
	fuBuffer []byte
	sps []byte
	pps []byte

	gotpkt bool
	pkt av.Packet
	timestamp uint32
}

func (self Stream) IsAudio() bool {
	return self.Sdp.AVType == "audio"
}

func (self Stream) IsVideo() bool {
	return self.Sdp.AVType == "video"
}

