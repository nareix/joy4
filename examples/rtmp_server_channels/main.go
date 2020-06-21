package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
)

func init() {
	format.RegisterAll()
}

type FrameDropper struct {
	Interval     int
	n            int
	skipping     bool
	DelaySkip    time.Duration
	lasttime     time.Time
	lastpkttime  time.Duration
	delay        time.Duration
	SkipInterval int
}

func (self *FrameDropper) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	if self.DelaySkip != 0 && pkt.Idx == int8(videoidx) {
		now := time.Now()
		if !self.lasttime.IsZero() {
			realdiff := now.Sub(self.lasttime)
			pktdiff := pkt.Time - self.lastpkttime
			self.delay += realdiff - pktdiff
		}
		self.lasttime = time.Now()
		self.lastpkttime = pkt.Time

		if !self.skipping {
			if self.delay > self.DelaySkip {
				self.skipping = true
				self.delay = 0
			}
		} else {
			if pkt.IsKeyFrame {
				self.skipping = false
			}
		}
		if self.skipping {
			drop = true
		}

		if self.SkipInterval != 0 && pkt.IsKeyFrame {
			if self.n == self.SkipInterval {
				self.n = 0
				self.skipping = true
			}
			self.n++
		}
	}

	if self.Interval != 0 {
		if self.n >= self.Interval && pkt.Idx == int8(videoidx) && !pkt.IsKeyFrame {
			drop = true
			self.n = 0
		}
		self.n++
	}

	return
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

			filters = append(filters, &pktque.FixTime{StartFromZero: true, MakeIncrement: true})

			if q := query.Get("framedrop"); q != "" {
				n := 0
				fmt.Sscanf(q, "%d", &n)
				filters = append(filters, &FrameDropper{Interval: n})
			}

			if q := query.Get("delayskip"); q != "" {
				dur, _ := time.ParseDuration(q)
				skipper := &FrameDropper{DelaySkip: dur}
				if q := query.Get("skipinterval"); q != "" {
					n := 0
					fmt.Sscanf(q, "%d", &n)
					skipper.SkipInterval = n
				}
				filters = append(filters, skipper)
			}

			demuxer := &pktque.FilterDemuxer{
				Filter:  filters,
				Demuxer: cursor,
			}

			avutil.CopyFile(conn, demuxer)
		}
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		l.Lock()
		ch := channels[conn.URL.Path]
		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			query := conn.URL.Query()
			if q := query.Get("cachegop"); q != "" {
				var n int
				fmt.Sscanf(q, "%d", &n)
				ch.que.SetMaxGopCount(n)
			}
			channels[conn.URL.Path] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			return
		}

		avutil.CopyFile(ch.que, conn)

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

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie?cachegop=2
	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie?cachegop=1
}
