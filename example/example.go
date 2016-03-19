
package main

import (
	"github.com/nareix/mp4"
	"os"
	"fmt"
	"encoding/hex"
)

func DemuxExample() {
	file, _ := os.Open("test.mp4")
	demuxer := &mp4.Demuxer{
		R: file,
	}
	demuxer.ReadHeader()

	fmt.Println("Total tracks: ", len(demuxer.Tracks))
	fmt.Println("Duration: ", demuxer.TrackH264.Duration())

	count := demuxer.TrackH264.SampleCount()
	fmt.Println("SampleCount: ", count)

	demuxer.TrackH264.SeekToTime(2.3)

	var sample []byte
	for i := 0; i < 5; i++ {
		pts, dts, isKeyFrame, data, err := demuxer.TrackH264.ReadSample()
		fmt.Println("sample #",
			i, pts, dts, isKeyFrame, len(data),
			demuxer.TrackH264.CurTime(),
			err,
		)
		if i == 3 {
			sample = data
		}
	}
	fmt.Println("Sample H264 frame:")
	fmt.Print(hex.Dump(sample))

	fmt.Println("Duration(AAC): ", demuxer.TrackAAC.Duration())
	fmt.Println("SampleCount(AAC): ", demuxer.TrackAAC.SampleCount())
	demuxer.TrackAAC.SeekToTime(1.3)

	for i := 0; i < 5; i++ {
		pts, dts, isKeyFrame, data, err := demuxer.TrackAAC.ReadSample()
		fmt.Println("sample(AAC) #",
			i, pts, dts, isKeyFrame, len(data),
			demuxer.TrackAAC.CurTime(),
			err,
		)
		if i == 1 {
			sample = data
		}
	}
	fmt.Println("Sample AAC frame:")
	fmt.Print(hex.Dump(sample))
}

func main() {
	DemuxExample()
}

