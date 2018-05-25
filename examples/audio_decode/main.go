
package main

import (
	"github.com/jinleileiking/joy4/av"
	"github.com/jinleileiking/joy4/format"
	"github.com/jinleileiking/joy4/av/avutil"
	"github.com/jinleileiking/joy4/cgo/ffmpeg"
)

// need ffmpeg installed

func init() {
	format.RegisterAll()
}

func main() {
	file, _ := avutil.Open("projectindex.flv")
	streams, _ := file.Streams()
	var dec *ffmpeg.AudioDecoder

	for _, stream := range streams {
		if stream.Type() == av.AAC {
			dec, _ = ffmpeg.NewAudioDecoder(stream.(av.AudioCodecData))
		}
	}

	for i := 0; i < 10; i++ {
		pkt, _ := file.ReadPacket()
		if streams[pkt.Idx].Type() == av.AAC {
			ok, frame, _ := dec.Decode(pkt.Data)
			if ok {
				println("decode samples", frame.SampleCount)
			}
		}
	}

	file.Close()
}

