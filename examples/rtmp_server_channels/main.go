package main

import (
	"sync"
	"fmt"
	"time"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/format/rtmp"
)

func init() {
	format.RegisterAll()
}

func main() {
	server := &rtmp.Server{}

	l := &sync.RWMutex{}
	type Channel struct {
		que *pubsub.Queue
	}
	channels := map[string]*Channel{}

	server.HandlePlay = func(conn *rtmp.Conn) {
		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		if ch != nil {
			cursor := ch.que.Latest()
			query := conn.URL.Query()
			if q := query.Get("delaygop"); q != "" {
				n := 0
				fmt.Sscanf(q, "%d", &n)
				cursor = ch.que.DelayedGopCount(n)
			} else if q := query.Get("delaytime"); q != "" {
				dur, _ := time.ParseDuration(q)
				cursor = ch.que.DelayedTime(dur)
			}
			filters := pktque.Filters{}
			if q := query.Get("waitkey"); q != "" {
				filters = append(filters, &pktque.WaitKeyFrame{})
			}
			filters = append(filters, &pktque.FixTime{StartFromZero: true})
			demuxer := &pktque.FilterDemuxer{
				Filter: filters,
				Demuxer: cursor,
			}
			avutil.CopyFile(conn, demuxer)
		}
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		streams, _ := conn.Streams()

		l.Lock()
		ch := channels[conn.URL.Path]
		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue(streams)
			query := conn.URL.Query()
			if q := query.Get("cachetime"); q != "" {
				dur, _ := time.ParseDuration(q)
				ch.que.SetMaxDuration(dur)
			}
			channels[conn.URL.Path] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			return
		}

		avutil.CopyPackets(ch.que, conn)

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()
		ch.que.Close()
	}

	server.ListenAndServe()

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie
	// ffmpeg -f avfoundation -i "0:0" .... -f flv rtmp://localhost/screen

	// with cache size options

	// ffplay rtmp://localhost/movie
	// ffplay rtmp://localhost/screen
	// ffplay rtmp://localhost/movie?delaytime=5s
	// ffplay rtmp://localhost/movie?delaytime=10s&waitkey=true
	// ffplay rtmp://localhost/movie?delaytime=20s

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie?cachetime=30s
	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie?cachetime=1m
}
