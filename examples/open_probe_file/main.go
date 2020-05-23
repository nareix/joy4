package main

import (
	"fmt"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/codec/h264parser"
)

func init() {
	format.RegisterAll()
}

func frameRate(info h264parser.SPSInfo) float64 {
	var num uint64
    var fps float64

    num   = 500 * uint64(info.TimeScale) /* 1000 * 0.5 */

	if info.UnitsInTick != 0 {
		fps = (float64(num) / float64(info.UnitsInTick)) / 1000
	}

    return fps;
}

func main() {
	file, _ := avutil.Open("/tmp/sintel.flv")

	if file == nil {
		fmt.Println("could not open input file")		
	}
	streams, _ := file.Streams()
	for _, stream := range streams {
		if stream.Type().IsAudio() {
			astream := stream.(av.AudioCodecData)
			fmt.Println(astream.Type(), astream.SampleRate(), astream.SampleFormat(), astream.ChannelLayout())
		} else if stream.Type().IsVideo() {
			vstream := stream.(av.VideoCodecData)
			fmt.Println(vstream.Type(), vstream.Width(), vstream.Height())
		}
	}

	for i := 0; i < 1000; i++ {
		var pkt av.Packet
		var err error
		if pkt, err = file.ReadPacket(); err != nil {
			break
		}

		// Split out the NAL units
		nals, _ := h264parser.SplitNALUs(pkt.Data)
		for _, nalUnit := range nals {

			if len(nalUnit) == 0 {
				continue
			}

			// Get the type, check for a SPS header
			typ := nalUnit[0] & 0x1f
			if typ == 7 {
				// Try to parse out the SPS header.
				if info, err := h264parser.ParseSPS(nalUnit); err == nil {
					fmt.Println("SPSInfo", 
						frameRate(info),
						info.UnitsInTick,
						info.TimeScale,
						info.FixedRate,
						info.Width)					
				}
			}
		}

		fmt.Println("pkt", i, streams[pkt.Idx].Type(), "len", len(pkt.Data), "keyframe", pkt.IsKeyFrame)
	}

	file.Close()
}

