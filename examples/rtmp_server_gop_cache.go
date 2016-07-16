package main

import (
	"fmt"
	"time"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
)

func init() {
	format.RegisterAll()
}

func main() {
	server := &rtmp.Server{}

	var que *pubsub.Queue

	go func() {
		file, _ := avutil.Open("projectindex.flv")
		streams, _ := file.Streams()
		que = pubsub.NewQueue(streams)
		demuxer := &pktque.FilterDemuxer{Demuxer: file, Filter: &pktque.Walltime{}}
		avutil.CopyPackets(que, demuxer)
		file.Close()
		que.Close()
	}()

	server.HandlePlay = func(conn *rtmp.Conn) {
		cursor := que.Latest()
		query := conn.URL.Query()
		if q := query.Get("delaygop"); q != "" {
			n := 0
			fmt.Sscanf(q, "%d", &n)
			cursor = que.DelayedGopCount(n)
		} else if q := query.Get("delaytime"); q != "" {
			dur, _ := time.ParseDuration(q)
			cursor = que.DelayedTime(dur)
		}
		demuxer := &pktque.FilterDemuxer{Demuxer: cursor, Filter: &pktque.WaitKeyFrame{}}
		avutil.CopyFile(conn, demuxer)
	}

	server.ListenAndServe()

	// ffplay rtmp://localhost/test.flv
	// ffplay rtmp://localhost/test.flv?delaygop=2
	// ffplay rtmp://localhost/test.flv?delaytime=3s
	// ffplay rtmp://localhost/test.flv?delaytime=10s
}

