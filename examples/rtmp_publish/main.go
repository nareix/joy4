package main

import (
	"flag"

	"github.com/tyrese/joy4/av/avutil"
	"github.com/tyrese/joy4/av/pktque"
	"github.com/tyrese/joy4/format"
	"github.com/tyrese/joy4/format/rtmp"
)

func init() {
	format.RegisterAll()
}

// as same as: ffmpeg -re -i projectindex.flv -c copy -f flv rtmp://localhost:1936/app/publish

func main() {
	inputFile := flag.String("i", "projectindex.flv", "input file")
	dstUrl := flag.String("o", "rtmp://localhost:1936/app/publish", "output url")
	debug := flag.Bool("v", false, "verbose")
	flag.Parse()
	rtmp.Debug = *debug
	file, _ := avutil.Open(*inputFile)
	conn, _ := rtmp.Dial(*dstUrl)
	// conn, _ := avutil.Create("rtmp://localhost:1936/app/publish")

	demuxer := &pktque.FilterDemuxer{Demuxer: file, Filter: &pktque.Walltime{}}
	avutil.CopyFile(conn, demuxer)

	file.Close()
	conn.Close()
}
