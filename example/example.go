
package main

import (
	"github.com/nareix/mp4"
	"os"
	"fmt"
)

func DemuxExample() {
	file, _ := os.Open("test.mp4")

	demuxer := &mp4.Demuxer{
		R: file,
	}
	demuxer.ReadHeader()

	fmt.Println("Duration: ", demuxer.TrackH264.Duration())
	count := demuxer.TrackH264.SampleCount()
	fmt.Println("SampleCount: ", count)
}

func main() {
	DemuxExample()
}

