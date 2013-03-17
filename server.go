
package rtmp

import (
	"bytes"
	"net"
	"fmt"
	"reflect"
	"io/ioutil"
	"os"
	"bufio"
	"log"
	"time"
	"strings"
)

var (
	event = make(chan eventS, 0)
	eventDone = make(chan int, 0)
)

type eventS struct {
	id int
	mr *MsgStream
	m *Msg
}

type eventID int

func (e eventS) String() string {
	switch e.id {
	case E_NEW:
		return "new"
	case E_PUBLISH:
		return "publish"
	case E_PLAY:
		return "play"
	case E_DATA:
		return fmt.Sprintf("data %d bytes ts %d", e.m.data.Len(), e.m.curts)
	case E_CLOSE:
		return "close"
	}
	return ""
}

/*
server:
 connect
 createStream
 publish
client:
 connect
 createStream
 getStreamLength
 play
*/

const (
	E_NEW = iota
	E_PUBLISH
	E_PLAY
	E_DATA
	E_CLOSE
)


func handleConnect(mr *MsgStream, trans float64, app string) {

	l.Printf("stream %v: connect: %s", mr, app)

	mr.app = app

	mr.WriteMsg32(2, MSG_ACK_SIZE, 0, 5000000)
	mr.WriteMsg32(2, MSG_BANDWIDTH, 0, 5000000)
	mr.WriteMsg32(2, MSG_CHUNK_SIZE, 0, 128)

	mr.WriteAMFCmd(3, 0, []AMFObj {
		AMFObj { atype : AMF_STRING, str : "_result", },
		AMFObj { atype : AMF_NUMBER, f64 : trans, },
		AMFObj { atype : AMF_OBJECT,
			obj : map[string] AMFObj {
				"fmtVer" : AMFObj { atype : AMF_STRING, str : "FMS/3,0,1,123", },
				"capabilities" : AMFObj { atype : AMF_NUMBER, f64 : 31, },
			},
		},
		AMFObj { atype : AMF_OBJECT,
			obj : map[string] AMFObj {
				"level" : AMFObj { atype : AMF_STRING, str : "status", },
				"code" : AMFObj { atype : AMF_STRING, str : "NetConnection.Connect.Success", },
				"description" : AMFObj { atype : AMF_STRING, str : "Connection Success.", },
				"objectEncoding" : AMFObj { atype : AMF_NUMBER, f64 : 0, },
			},
		},
	})
}

func handleMeta(mr *MsgStream, obj AMFObj) {

	mr.W = int(obj.obj["width"].f64)
	mr.H = int(obj.obj["height"].f64)

	l.Printf("stream %v: meta video %dx%d", mr, mr.W, mr.H)
}

func handleCreateStream(mr *MsgStream, trans float64) {

	l.Printf("stream %v: createStream", mr)

	mr.WriteAMFCmd(3, 0, []AMFObj {
		AMFObj { atype : AMF_STRING, str : "_result", },
		AMFObj { atype : AMF_NUMBER, f64 : trans, },
		AMFObj { atype : AMF_NULL, },
		AMFObj { atype : AMF_NUMBER, f64 : 1 },
	})
}

func handleGetStreamLength(mr *MsgStream, trans float64) {
}

func handlePublish(mr *MsgStream) {

	l.Printf("stream %v: publish", mr)

	mr.WriteAMFCmd(3, 0, []AMFObj {
		AMFObj { atype : AMF_STRING, str : "onStatus", },
		AMFObj { atype : AMF_NUMBER, f64 : 0, },
		AMFObj { atype : AMF_NULL, },
		AMFObj { atype : AMF_OBJECT,
			obj : map[string] AMFObj {
				"level" : AMFObj { atype : AMF_STRING, str : "status", },
				"code" : AMFObj { atype : AMF_STRING, str : "NetStream.Publish.Start", },
				"description" : AMFObj { atype : AMF_STRING, str : "Start publising.", },
			},
		},
	})

	event <- eventS{id:E_PUBLISH, mr:mr}
	<-eventDone
}

type testsrc struct {
	r *bufio.Reader
	dir string
	w,h int
	ts int
	codec string
	key bool
	idx int
	data []byte
}

func tsrcNew() (m *testsrc) {
	m = &testsrc{}
	m.dir = "/pixies/go/data/tmp"
	fi, _ := os.Open(fmt.Sprintf("%s/index", m.dir))
	m.r = bufio.NewReader(fi)
	l, _ := m.r.ReadString('\n')
	fmt.Sscanf(l, "%dx%d", &m.w, &m.h)
	return
}

func (m *testsrc) fetch() (err error) {
	l, err := m.r.ReadString('\n')
	if err != nil {
		return
	}
	a := strings.Split(l, ",")
	fmt.Sscanf(a[0], "%d", &m.ts)
	m.codec = a[1]
	fmt.Sscanf(a[2], "%d", &m.idx)
	switch m.codec {
	case "h264":
		fmt.Sscanf(a[3], "%t", &m.key)
		m.data, err = ioutil.ReadFile(fmt.Sprintf("%s/h264/%d.264", m.dir, m.idx))
	case "aac":
		m.data, err = ioutil.ReadFile(fmt.Sprintf("%s/aac/%d.aac", m.dir, m.idx))
	}
	return
}

func handlePlay(mr *MsgStream, strid int) {

	l.Printf("stream %v: play", mr)

	var tsrc *testsrc
	//tsrc = tsrcNew()

	if tsrc == nil {
		event <- eventS{id:E_PLAY, mr:mr}
		<-eventDone
	} else {
		l.Printf("stream %v: test play data in %s", mr, tsrc.dir)
		mr.W = tsrc.w
		mr.H = tsrc.h
		l.Printf("stream %v: test video %dx%d", mr, mr.W, mr.H)
	}

	var b bytes.Buffer
	WriteInt(&b, 0, 2)
	WriteInt(&b, strid, 4)
	mr.WriteMsg(0, 2, MSG_USER, 0, 0, b.Bytes()) // stream begin 1

	mr.WriteAMFCmd(5, strid, []AMFObj {
		AMFObj { atype : AMF_STRING, str : "onStatus", },
		AMFObj { atype : AMF_NUMBER, f64 : 0, },
		AMFObj { atype : AMF_NULL, },
		AMFObj { atype : AMF_OBJECT,
			obj : map[string] AMFObj {
				"level" : AMFObj { atype : AMF_STRING, str : "status", },
				"code" : AMFObj { atype : AMF_STRING, str : "NetStream.Play.Start", },
				"description" : AMFObj { atype : AMF_STRING, str : "Start live.", },
			},
		},
	})

	l.Printf("stream %v: video size %dx%d", mr, mr.W, mr.H)

	mr.WriteAMFMeta(5, strid, []AMFObj {
		AMFObj { atype : AMF_STRING, str : "|RtmpSampleAccess", },
		AMFObj { atype : AMF_BOOLEAN, i: 1, },
		AMFObj { atype : AMF_BOOLEAN, i: 1, },
	})

	mr.WriteAMFMeta(5, strid, []AMFObj {
		AMFObj { atype : AMF_STRING, str : "onMetaData", },
		AMFObj { atype : AMF_OBJECT,
			obj : map[string] AMFObj {
				"Server" : AMFObj { atype : AMF_STRING, str : "Golang Rtmp Server", },
				"width" : AMFObj { atype : AMF_NUMBER, f64 : float64(mr.W), },
				"height" : AMFObj { atype : AMF_NUMBER, f64 : float64(mr.H), },
				"displayWidth" : AMFObj { atype : AMF_NUMBER, f64 : float64(mr.W), },
				"displayHeight" : AMFObj { atype : AMF_NUMBER, f64 : float64(mr.H), },
				"duration" : AMFObj { atype : AMF_NUMBER, f64 : 0, },
				"framerate" : AMFObj { atype : AMF_NUMBER, f64 : 25000, },
				"videodatarate" : AMFObj { atype : AMF_NUMBER, f64 : 731, },
				"videocodecid" : AMFObj { atype : AMF_NUMBER, f64 : 7, },
				"audiodatarate" : AMFObj { atype : AMF_NUMBER, f64 : 122, },
				"audiocodecid" : AMFObj { atype : AMF_NUMBER, f64 : 10, },
			},
		},
	})

	if tsrc == nil {
		l.Printf("stream %v: extra size %d %d", mr, len(mr.extraA), len(mr.extraV))

		mr.WriteAAC(strid, 0, mr.extraA[2:])
		mr.WritePPS(strid, 0, mr.extraV[5:])

		l.Printf("stream %v: player wait data", mr)

		for {
			m := <-mr.que
			l.Printf("data %v: got %v", mr, m)
			switch m.typeid {
			case MSG_AUDIO:
				mr.WriteAudio(strid, m.curts, m.data.Bytes()[2:])
			case MSG_VIDEO:
				mr.WriteVideo(strid, m.curts, m.key, m.data.Bytes()[5:])
			}
		}
	} else {

		lf, _ := os.Create("/tmp/rtmp.log")
		ll := log.New(lf, "", 0)

		starttm := time.Now()
		k := 0

		for {
			err := tsrc.fetch()
			if err != nil {
				panic(err)
			}
			switch tsrc.codec {
			case "h264":
				if tsrc.idx == 0 {
					mr.WritePPS(strid, 0, tsrc.data)
				} else {
					mr.WriteVideo(strid, tsrc.ts, tsrc.key, tsrc.data)
				}
			case "aac":
				if tsrc.idx == 0 {
					mr.WriteAAC(strid, 0, tsrc.data)
				} else {
					mr.WriteAudio(strid, tsrc.ts, tsrc.data)
				}
			}
			dur := time.Since(starttm).Nanoseconds()
			diff := tsrc.ts - 1000 - int(dur/1000000)
			if diff > 0 {
				time.Sleep(time.Duration(diff)*time.Millisecond)
			}
			l.Printf("data %v: ts %v dur %v diff %v", mr, tsrc.ts, int(dur/1000000), diff)
			ll.Printf("#%d %d,%s,%d %d", k, tsrc.ts, tsrc.codec, tsrc.idx, len(tsrc.data))
			k++
		}
	}
}

func serve(mr *MsgStream) {

	defer func() {
		if err := recover(); err != nil {
			event <- eventS{id:E_CLOSE, mr:mr}
			<-eventDone
			l.Printf("stream %v: closed %v", mr, err)
		}
	}()

	handShake(mr.r)

//	f, _ := os.Create("/tmp/pub.log")
//	mr.l = log.New(f, "", 0)

	for {
		m := mr.ReadMsg()
		if m == nil {
			continue
		}

		//l.Printf("stream %v: msg %v", mr, m)

		if m.typeid == MSG_AUDIO || m.typeid == MSG_VIDEO {
//			mr.l.Printf("%d,%d", m.typeid, m.data.Len())
			event <- eventS{id:E_DATA, mr:mr, m:m}
			<-eventDone
		}

		if m.typeid == MSG_AMF_CMD || m.typeid == MSG_AMF_META {
			a := ReadAMF(m.data)
			//l.Printf("server: amfobj %v\n", a)
			switch a.str {
			case "connect":
				a2 := ReadAMF(m.data)
				a3 := ReadAMF(m.data)
				if _, ok := a3.obj["app"]; !ok || a3.obj["app"].str == "" {
					panic("connect: app not found")
				}
				handleConnect(mr, a2.f64, a3.obj["app"].str)
			case "@setDataFrame":
				ReadAMF(m.data)
				a3 := ReadAMF(m.data)
				handleMeta(mr, a3)
				l.Printf("stream %v: setdataframe", mr)
			case "createStream":
				a2 := ReadAMF(m.data)
				handleCreateStream(mr, a2.f64)
			case "publish":
				handlePublish(mr)
			case "play":
				handlePlay(mr, m.strid)
			}
		}
	}
}

func listenEvent() {
	idmap := map[string]*MsgStream{}
	pubmap := map[string]*MsgStream{}

	for {
		e := <-event
		if e.id == E_DATA {
			l.Printf("data %v: %v", e.mr, e)
		} else {
			l.Printf("event %v: %v", e.mr, e)
		}
		switch {
		case e.id == E_NEW:
			idmap[e.mr.id] = e.mr
		case e.id == E_PUBLISH:
			if _, ok := pubmap[e.mr.app]; ok {
				l.Printf("event %v: duplicated publish with %v app %s", e.mr, pubmap[e.mr.app], e.mr.app)
				e.mr.Close()
			} else {
				e.mr.role = PUBLISHER
				pubmap[e.mr.app] = e.mr
			}
		case e.id == E_PLAY:
			src, ok := pubmap[e.mr.app]
			if !ok || src.stat != WAIT_DATA {
				l.Printf("event %v: cannot find publisher with app %s", e.mr, e.mr.app)
				e.mr.Close()
			} else {
				e.mr.W = src.W
				e.mr.H = src.H
				e.mr.role = PLAYER
				e.mr.extraA = src.extraA
				e.mr.extraV = src.extraV
				e.mr.que = make(chan *Msg, 16)
			}
		case e.id == E_CLOSE:
			if e.mr.role == PUBLISHER {
				delete(pubmap, e.mr.app)
			}
			delete(idmap, e.mr.id)
		case e.id == E_DATA && e.mr.stat == WAIT_EXTRA:
			if len(e.mr.extraA) == 0 && e.m.typeid == MSG_AUDIO {
				l.Printf("event %v: got aac config", e.mr)
				e.mr.extraA = e.m.data.Bytes()
			}
			if len(e.mr.extraV) == 0 && e.m.typeid == MSG_VIDEO {
				l.Printf("event %v: got pps", e.mr)
				e.mr.extraV = e.m.data.Bytes()
			}
			if len(e.mr.extraA) > 0 && len(e.mr.extraV) > 0 {
				l.Printf("event %v: got all extra", e.mr)
				e.mr.stat = WAIT_DATA
			}
		case e.id == E_DATA && e.mr.stat == WAIT_DATA:
			for _, mr := range idmap {
				if mr.role == PLAYER && mr.app == e.mr.app {
					ch := reflect.ValueOf(mr.que)
					ok := ch.TrySend(reflect.ValueOf(e.m))
					if !ok {
						l.Printf("event %v: send failed", e.mr)
					} else {
						l.Printf("event %v: send ok", e.mr)
					}
				}
			}
		}
		eventDone <- 1
	}
}

func SimpleServer() {
	l.Printf("server: simple server starts")
	ln, err := net.Listen("tcp", ":1935")
	if err != nil {
		l.Printf("server: error: listen 1935 %s\n", err)
		return
	}
	go listenEvent()
	for {
		c, err := ln.Accept()
		if err != nil {
			l.Printf("server: error: sock accept %s\n", err)
			break
		}
		go func (c net.Conn) {
			mr := NewMsgStream(c)
			event <- eventS{id:E_NEW, mr:mr}
			<-eventDone
			serve(mr)
		} (c)
	}
}

