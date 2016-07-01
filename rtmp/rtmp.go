
package rtmp

import (
	"strings"
	"bytes"
	"net"
	"net/url"
	"bufio"
	"time"
	"fmt"
	"encoding/hex"
	"io"
	"github.com/nareix/pio"
	"github.com/nareix/joy4/flv/flvio"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/codec"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/codec/aacparser"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/rand"
)

var MaxProbePacketCount = 6

func ParseURL(uri string) (host string, path string) {
	var u *url.URL
	if u, _ = url.Parse(uri); u == nil {
		return
	}
	host = u.Host
	if _, _, err := net.SplitHostPort(u.Host); err != nil {
		host += ":1935"
	}
	path = u.Path
	return
}

func Dial(uri string) (conn *Conn, err error) {
	return DialTimeout(uri, 0)
}

func DialTimeout(uri string, timeout time.Duration) (conn *Conn, err error) {
	host, path := ParseURL(uri)
	dailer := net.Dialer{Timeout: timeout}
	var netconn net.Conn
	if netconn, err = dailer.Dial("tcp", host); err != nil {
		return
	}

	conn = NewConn(netconn)
	conn.Host = host
	conn.Path = path
	return
}

type Server struct {
	Debug bool
	DebugConn bool
	Addr string
	HandlePublish func(*Conn)
	HandlePlay func(*Conn)
}

func (self *Server) handleConn(conn *Conn) (err error) {
	if err = conn.prepare(stageCommandDone, 0); err != nil {
		return
	}

	if conn.playing {
		if self.HandlePlay != nil {
			self.HandlePlay(conn)
		}
	} else if conn.publishing {
		if self.HandlePublish != nil {
			self.HandlePublish(conn)
		}
	}

	if err = conn.Close(); err != nil {
		return
	}

	return
}

func (self *Server) ListenAndServe() (err error) {
	addr := self.Addr
	if addr == "" {
		addr = ":1935"
	}
	var tcpaddr *net.TCPAddr
	if tcpaddr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
		err = fmt.Errorf("rtmp: ListenAndServe: %s", err)
		return
	}

	var listener *net.TCPListener
	if listener, err = net.ListenTCP("tcp", tcpaddr); err != nil {
		return
	}

	if self.Debug {
		fmt.Println("rtmp: server: listening on", addr)
	}

	var netconn net.Conn
	for {
		if netconn, err = listener.Accept(); err != nil {
			return
		}

		if self.Debug {
			fmt.Println("rtmp: server: accepted")
		}

		conn := NewConn(netconn)
		conn.Debug = self.DebugConn
		conn.isserver = true
		go func() {
			err = self.handleConn(conn)
			if self.Debug {
				fmt.Println("rtmp: server: client closed err:", err)
			}
		}()
	}
}

type stream struct {
	codec av.CodecData
	lasttime time.Duration
}

func newStream(codec av.CodecData) *stream {
	return &stream{
		codec: codec,
	}
}

const (
	stageHandshakeDone = iota+1
	stageCommandDone
	stageCodecDataDone
)

const (
	prepareReading = iota+1
	prepareWriting
)

type Conn struct {
	Host string
	Path string
	Debug bool

	streams []*stream
	videostreamidx, audiostreamidx int

	probepkts []flvio.Tag

	br *pio.Reader
	bw *pio.Writer
	bufr *bufio.Reader
	bufw *bufio.Writer
	intw *pio.Writer
	netconn net.Conn

	writeMaxChunkSize int
	readMaxChunkSize int
	readcsmap map[uint32]*chunkStream
	writecsmap map[uint32]*chunkStream

	isserver bool
	publishing, playing bool
	reading, writing bool
	stage int

	avmsgsid uint32

	gotcommand bool
	commandname string
	commandtransid float64
	commandobj flvio.AMFMap
	commandparams []interface{}

	gotmsg bool
	timestamp uint32
	msgdata []byte
	msgtypeid uint8
	datamsgvals []interface{}
	videodata *flvio.Videodata
	audiodata *flvio.Audiodata

	eventtype uint16
}

func NewConn(netconn net.Conn) *Conn {
	conn := &Conn{}
	conn.netconn = netconn
	conn.readcsmap = make(map[uint32]*chunkStream)
	conn.writecsmap = make(map[uint32]*chunkStream)
	conn.readMaxChunkSize = 128
	conn.writeMaxChunkSize = 128
	conn.bufr = bufio.NewReaderSize(netconn, 2048)
	conn.bufw = bufio.NewWriterSize(netconn, 2048)
	conn.br = pio.NewReader(conn.bufr)
	conn.bw = pio.NewWriter(conn.bufw)
	conn.intw = pio.NewWriter(nil)
	return conn
}

type chunkStream struct {
	timenow uint32
	timedelta uint32
	hastimeext bool
	msgsid uint32
	msgtypeid uint8
	msgdatalen uint32
	msgdataleft uint32
	msghdrtype uint8
	msgdata []byte
}

func (self *chunkStream) Start() {
	self.msgdataleft = self.msgdatalen
	self.msgdata = make([]byte, self.msgdatalen)
}

const (
	msgtypeidUserControl = 4
	msgtypeidWindowAckSize = 5
	msgtypeidSetPeerBandwidth = 6
	msgtypeidSetChunkSize = 1
	msgtypeidCommandMsgAMF0 = 20
	msgtypeidCommandMsgAMF3 = 17
	msgtypeidDataMsgAMF0 = 18
	msgtypeidDataMsgAMF3 = 15
	msgtypeidVideoMsg = 9
	msgtypeidAudioMsg = 8
)

const (
	eventtypeStreamBegin = 0
	eventtypeSetBufferLength = 3
	eventtypeStreamIsRecorded = 4
)

func (self *Conn) Close() (err error) {
	return self.netconn.Close()
}

func (self *Conn) pollCommand() (err error) {
	for {
		if err = self.pollMsg(); err != nil {
			return
		}
		if self.gotcommand {
			return
		}
	}
}

func (self *Conn) pollMsg() (err error) {
	self.gotmsg = false
	self.gotcommand = false
	self.datamsgvals = nil
	self.videodata = nil
	self.audiodata = nil
	for {
		if err = self.readChunk(); err != nil {
			return
		}
		if self.gotmsg {
			return
		}
	}
}

func splitPath(s string) (app, play string) {
	pathsegs := strings.Split(s, "/")
	if len(pathsegs) > 1 {
		app = pathsegs[1]
	}
	if len(pathsegs) > 2 {
		play = pathsegs[2]
	}
	return
}

func formatPath(app, play string) string {
	ps := strings.Split(app+"/"+play, "/")
	out := []string{""}
	for _, s := range ps {
		if len(s) > 0 {
			out = append(out, s)
		}
	}
	return strings.Join(out, "/")
}

func (self *Conn) recvConnect() (err error) {
	var connectpath string

	// < connect("app")
	if err = self.pollCommand(); err != nil {
		return
	}
	if self.commandname != "connect" {
		err = fmt.Errorf("rtmp: first command is not connect")
		return
	}
	if self.commandobj != nil {
		app := self.commandobj["app"]
		connectpath, _ = app.(string)
	}

	// > WindowAckSize
	if err = self.writeWindowAckSize(5000000); err != nil {
		return
	}
	// > SetPeerBandwidth
	if err = self.writeSetPeerBandwidth(5000000, 2); err != nil {
		return
	}
	self.writeMaxChunkSize = 1024*1024*128
	// > SetChunkSize
	if err = self.writeSetChunkSize(uint32(self.writeMaxChunkSize)); err != nil {
		return
	}

	// > _result("NetConnection.Connect.Success")
	w := self.writeCommandMsgStart()
	flvio.WriteAMF0Val(w, "_result")
	flvio.WriteAMF0Val(w, self.commandtransid)
	flvio.WriteAMF0Val(w, flvio.AMFMap{
		"fmtVer": "FMS/3,0,1,123",
		"capabilities": 31,
	})
	flvio.WriteAMF0Val(w, flvio.AMFMap{
		"level": "status",
		"code": "NetConnection.Connect.Success",
		"description": "Connection succeeded.",
		"objectEncoding": 3,
	})
	if err = self.writeCommandMsgEnd(3, 0); err != nil {
		return
	}

	for {
		if err = self.pollMsg(); err != nil {
			return
		}
		if self.gotcommand {
			switch self.commandname {

			// < createStream
			case "createStream":
				self.avmsgsid = uint32(1)
				// > _result(streamid)
				w := self.writeCommandMsgStart()
				flvio.WriteAMF0Val(w, "_result")
				flvio.WriteAMF0Val(w, self.commandtransid)
				flvio.WriteAMF0Val(w, nil)
				flvio.WriteAMF0Val(w, self.avmsgsid) // streamid=1
				if err = self.writeCommandMsgEnd(3, 0); err != nil {
					return
				}

			// < publish("path")
			case "publish":
				if self.Debug {
					fmt.Println("rtmp: < publish")
				}

				if len(self.commandparams) < 1 {
					err = fmt.Errorf("rtmp: publish params invalid")
					return
				}
				publishpath, _ := self.commandparams[0].(string)

				// > onStatus()
				w := self.writeCommandMsgStart()
				flvio.WriteAMF0Val(w, "onStatus")
				flvio.WriteAMF0Val(w, self.commandtransid)
				flvio.WriteAMF0Val(w, nil)
				flvio.WriteAMF0Val(w, flvio.AMFMap{
					"level": "status",
					"code": "NetStream.Publish.Start",
					"description": "Start publishing",
				})
				if err = self.writeCommandMsgEnd(5, self.avmsgsid); err != nil {
					return
				}

				self.Path = formatPath(connectpath, publishpath)
				self.publishing = true
				self.reading = true
				self.stage++
				return

			// < play("path")
			case "play":
				if self.Debug {
					fmt.Println("rtmp: < play")
				}

				if len(self.commandparams) < 1 {
					err = fmt.Errorf("rtmp: command play params invalid")
					return
				}
				playpath, _ := self.commandparams[0].(string)

				// > streamBegin(streamid)
				if err = self.writeStreamBegin(self.avmsgsid); err != nil {
					return
				}

				// > onStatus()
				w := self.writeCommandMsgStart()
				flvio.WriteAMF0Val(w, "onStatus")
				flvio.WriteAMF0Val(w, self.commandtransid)
				flvio.WriteAMF0Val(w, nil)
				flvio.WriteAMF0Val(w, flvio.AMFMap{
					"level": "status",
					"code": "NetStream.Play.Start",
					"description": "Start live",
				})
				if err = self.writeCommandMsgEnd(5, self.avmsgsid); err != nil {
					return
				}

				// > |RtmpSampleAccess()
				w = self.writeDataMsgStart()
				flvio.WriteAMF0Val(w, "|RtmpSampleAccess")
				flvio.WriteAMF0Val(w, true)
				flvio.WriteAMF0Val(w, true)
				if err = self.writeDataMsgEnd(5, self.avmsgsid); err != nil {
					return
				}

				self.Path = formatPath(connectpath, playpath)
				self.playing = true
				self.writing = true
				self.stage++
				return
			}

		}
	}

	return
}

func (self *Conn) checkConnectResult() (ok bool, errmsg string) {
	errmsg = "params invalid"
	if len(self.commandparams) < 1 {
		return
	}

	obj, _ := self.commandparams[0].(flvio.AMFMap)
	if obj == nil {
		return
	}

	_code, _ := obj["code"]
	if _code == nil {
		return
	}
	code, _ := _code.(string)
	if code != "NetConnection.Connect.Success" {
		return
	}

	ok = true
	return
}

func (self *Conn) checkCreateStreamResult() (ok bool, avmsgsid uint32) {
	if len(self.commandparams) < 1 {
		return
	}

	ok = true
	_avmsgsid, _ := self.commandparams[0].(float64)
	avmsgsid = uint32(_avmsgsid)
	return
}

func (self *Conn) probe() (err error) {
	for i := 0; i < MaxProbePacketCount; {
		if err = self.pollMsg(); err != nil {
			return
		}

		switch self.msgtypeid {
		case msgtypeidVideoMsg:
			i++
			tag := self.videodata
			switch tag.CodecID {
			case flvio.VIDEO_H264:
				if tag.AVCPacketType == flvio.AVC_SEQHDR {
					var h264 h264parser.CodecData
					if h264, err = h264parser.NewCodecDataFromAVCDecoderConfRecord(tag.Data); err != nil {
						err = fmt.Errorf("rtmp: h264 codec data invalid")
						return
					}
					self.videostreamidx = len(self.streams)
					self.streams = append(self.streams, newStream(h264))
				} else {
					self.probepkts = append(self.probepkts, tag)
				}

			default:
				err = fmt.Errorf("rtmp: video CodecID=%d not supported", tag.CodecID)
				return
			}

		case msgtypeidAudioMsg:
			i++
			tag := self.audiodata
			switch tag.SoundFormat {
			case flvio.SOUND_AAC:
				if tag.AACPacketType == flvio.AAC_SEQHDR {
					var aac aacparser.CodecData
					if aac, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(tag.Data); err != nil {
						err = fmt.Errorf("rtmp: aac codec data invalid")
						return
					}
					self.audiostreamidx = len(self.streams)
					self.streams = append(self.streams, newStream(aac))
				} else {
					self.probepkts = append(self.probepkts, tag)
				}

			case flvio.SOUND_NELLYMOSER, flvio.SOUND_NELLYMOSER_16KHZ_MONO, flvio.SOUND_NELLYMOSER_8KHZ_MONO:
				stream := codec.NewNellyMoserCodecData()
				self.audiostreamidx = len(self.streams)
				self.streams = append(self.streams, newStream(stream))
				self.probepkts = append(self.probepkts, tag)

			case flvio.SOUND_ALAW:
				stream := codec.NewPCMAlawCodecData()
				self.audiostreamidx = len(self.streams)
				self.streams = append(self.streams, newStream(stream))
				self.probepkts = append(self.probepkts, tag)

			case flvio.SOUND_MULAW:
				stream := codec.NewPCMMulawCodecData()
				self.audiostreamidx = len(self.streams)
				self.streams = append(self.streams, newStream(stream))
				self.probepkts = append(self.probepkts, tag)

			case flvio.SOUND_SPEEX:
				stream := codec.NewSpeexCodecData()
				self.audiostreamidx = len(self.streams)
				self.streams = append(self.streams, newStream(stream))
				self.probepkts = append(self.probepkts, tag)

			default:
				err = fmt.Errorf("rtmp: audio SoundFormat=%d not supported", tag.SoundFormat)
				return
			}
		}

		if len(self.streams) == 2 {
			break
		}
	}

	if len(self.streams) == 0 {
		err = fmt.Errorf("rtmp: probe failed")
		return
	}

	self.stage++
	return
}

func (self *Conn) connectPlay() (err error) {
	connectpath, playpath := splitPath(self.Path)

	// > connect("app")
	if self.Debug {
		fmt.Printf("rtmp: > connect('%s') host=%s\n", connectpath, self.Host)
	}
	w := self.writeCommandMsgStart()
	flvio.WriteAMF0Val(w, "connect")
	flvio.WriteAMF0Val(w, 1)
	flvio.WriteAMF0Val(w, flvio.AMFMap{
		"app": connectpath,
		"flashVer": "MAC 22,0,0,192",
		"tcUrl": fmt.Sprintf("rtmp://%s/%s", self.Host, connectpath),
		"fpad": false,
		"capabilities": 15,
		"audioCodecs": 4071,
		"videoCodecs": 252,
		"videoFunction": 1,
	})
	if err = self.writeCommandMsgEnd(3, 0); err != nil {
		return
	}

	for {
		if err = self.pollMsg(); err != nil {
			return
		}
		if self.gotcommand {
			// < _result("NetConnection.Connect.Success")
			if self.commandname == "_result" {
				var ok bool
				var errmsg string
				if ok, errmsg = self.checkConnectResult(); !ok {
					err = fmt.Errorf("rtmp: command connect failed: %s", errmsg)
					return
				}
				if self.Debug {
					fmt.Printf("rtmp: < _result() of connect\n")
				}
				break
			}
		} else {
			if self.msgtypeid == msgtypeidWindowAckSize {
				if err = self.writeWindowAckSize(2500000); err != nil {
					return
				}
			}
		}
	}

	// > createStream()
	if self.Debug {
		fmt.Printf("rtmp: > createStream()\n")
	}
	w = self.writeCommandMsgStart()
	flvio.WriteAMF0Val(w, "createStream")
	flvio.WriteAMF0Val(w, 2)
	flvio.WriteAMF0Val(w, nil)
	if err = self.writeCommandMsgEnd(3, 0); err != nil {
		return
	}

	// > SetBufferLength 0,100ms
	if err = self.writeSetBufferLength(0, 100); err != nil {
		return
	}

	for {
		if err = self.pollMsg(); err != nil {
			return
		}
		if self.gotcommand {
			// < _result(avmsgsid) of createStream
			if self.commandname == "_result" {
				var ok bool
				if ok, self.avmsgsid = self.checkCreateStreamResult(); !ok {
					err = fmt.Errorf("rtmp: createStream command failed")
					return
				}
				break
			}
		}
	}

	// > play('app')
	if self.Debug {
		fmt.Printf("rtmp: > play('%s')\n", playpath)
	}
	w = self.writeCommandMsgStart()
	flvio.WriteAMF0Val(w, "play")
	flvio.WriteAMF0Val(w, 0)
	flvio.WriteAMF0Val(w, nil)
	flvio.WriteAMF0Val(w, playpath)
	if err = self.writeCommandMsgEnd(8, self.avmsgsid); err != nil {
		return
	}

	self.reading = true
	self.playing = true
	self.stage++
	return
}

func (self *Conn) ReadPacket() (pkt av.Packet, err error) {
	if err = self.prepare(stageCodecDataDone, prepareReading); err != nil {
		return
	}

	poll: for {
		var _tag flvio.Tag

		if len(self.probepkts) > 0 {
			_tag = self.probepkts[0]
			self.probepkts = self.probepkts[1:]
		} else {
			if err = self.pollMsg(); err != nil {
				return
			}
			switch self.msgtypeid {
			case msgtypeidVideoMsg:
				_tag = self.videodata
			case msgtypeidAudioMsg:
				_tag = self.audiodata
			}
		}

		switch tag := _tag.(type) {
		case *flvio.Videodata:
			pkt.CompositionTime = tsToTime(uint32(tag.CompositionTime))
			var ok bool
			if pkt.Data, ok = h264parser.FindDataNALUInAVCCNALUs(tag.Data); !ok {
				err = fmt.Errorf("rtmp: input h264 format invalid")
				return
			}
			pkt.IsKeyFrame = tag.FrameType == flvio.FRAME_KEY
			pkt.Idx = int8(self.videostreamidx)
			break poll

		case *flvio.Audiodata:
			pkt.Data = tag.Data
			pkt.Idx = int8(self.audiostreamidx)
			break poll
		}
	}

	pkt.Time = tsToTime(self.timestamp)
	return
}

func (self *Conn) prepare(stage int, flags int) (err error) {
	for self.stage < stage {
		switch self.stage {
		case 0:
			if self.isserver {
				if err = self.handshakeServer(); err != nil {
					return
				}
			} else {
				if err = self.handshakeClient(); err != nil {
					return
				}
			}

		case stageHandshakeDone:
			if self.isserver {
				if err = self.recvConnect(); err != nil {
					return
				}
			} else {
				if flags == prepareReading {
					if err = self.connectPlay(); err != nil {
						return
					}
				} else {
					err = fmt.Errorf("rtmp: connect publish unsupported")
					return
				}
			}

		case stageCommandDone:
			if flags == prepareReading {
				if err = self.probe(); err != nil {
					return
				}
			} else {
				err = fmt.Errorf("rtmp: call WriteHeader() before WritePacket()")
				return
			}
		}
	}
	return
}

func (self *Conn) Streams() (streams []av.CodecData, err error) {
	if err = self.prepare(stageCodecDataDone, prepareReading); err != nil {
		return
	}
	for _, stream := range self.streams {
		streams = append(streams, stream.codec)
	}
	return
}

func timeToTs(tm time.Duration) uint32 {
	return uint32(tm / time.Millisecond)
}

func tsToTime(ts uint32) time.Duration {
	return time.Duration(ts)*time.Millisecond
}

func (self *Conn) WritePacket(pkt av.Packet) (err error) {
	if err = self.prepare(stageCodecDataDone, prepareWriting); err != nil {
		return
	}

	stream := self.streams[pkt.Idx]
	ts := timeToTs(pkt.Time)

	if self.Debug {
		fmt.Println("rtmp: WritePacket", pkt.Idx, pkt.Time, pkt.CompositionTime, ts)
	}

	codec := stream.codec
	switch codec.Type() {
	case av.AAC:
		audiodata := self.makeAACAudiodata(codec.(av.AudioCodecData), flvio.AAC_RAW, pkt.Data)
		w := self.writeAudioDataStart()
		audiodata.Marshal(w)
		if err = self.writeAudioDataEnd(ts); err != nil {
			return
		}

	case av.H264:
		if typ := h264parser.CheckNALUsType(pkt.Data); typ != h264parser.NALU_RAW {
			err = fmt.Errorf("rtmp: h264 nalu format=%d invalid", typ)
			return
		}
		var b [4]byte
		pio.PutU32BE(b[:], uint32(len(pkt.Data)))
		videodata := self.makeH264Videodata(flvio.AVC_NALU, pkt.IsKeyFrame, []byte{})
		videodata.Datav = [][]byte{b[:], pkt.Data}
		videodata.CompositionTime = int32(timeToTs(pkt.CompositionTime))
		w := self.writeVideoDataStart()
		videodata.Marshal(w)
		if err = self.writeVideoDataEnd(ts); err != nil {
			return
		}
	}

	return
}

func (self *Conn) WriteHeader(streams []av.CodecData) (err error) {
	if err = self.prepare(stageCommandDone, prepareWriting); err != nil {
		return
	}

	metadata := flvio.AMFMap{}

	for _, _stream := range streams {
		typ := _stream.Type()
		switch {
		case typ.IsVideo():
			stream := _stream.(av.VideoCodecData)
			switch typ {
			case av.H264:
				metadata["videocodecid"] = flvio.VIDEO_H264

			default:
				err = fmt.Errorf("rtmp: WriteHeader: unsupported video codecType=%v", stream.Type())
				return
			}

			metadata["width"] = stream.Width()
			metadata["height"] = stream.Height()
			metadata["displayWidth"] = stream.Width()
			metadata["displayHeight"] = stream.Height()

		case typ.IsAudio():
			stream := _stream.(av.AudioCodecData)
			switch typ {
			case av.AAC:
				metadata["audiocodecid"] = flvio.SOUND_AAC

			default:
				err = fmt.Errorf("rtmp: WriteHeader: unsupported audio codecType=%v", stream.Type())
				return
			}

			metadata["audiosamplerate"] = stream.SampleRate()
		}
	}

	// > onMetaData()
	w := self.writeDataMsgStart()
	flvio.WriteAMF0Val(w, "onMetaData")
	flvio.WriteAMF0Val(w, metadata)
	if err = self.writeDataMsgEnd(5, self.avmsgsid); err != nil {
		return
	}

	// > Videodata(decoder config)
	// > Audiodata(decoder config)
	for _, stream := range streams {
		switch stream.Type() {
		case av.H264:
			h264 := stream.(h264parser.CodecData)
			videodata := self.makeH264Videodata(flvio.AVC_SEQHDR, true, h264.AVCDecoderConfRecordBytes())
			w := self.writeVideoDataStart()
			videodata.Marshal(w)
			if err = self.writeVideoDataEnd(0); err != nil {
				return
			}
			self.streams = append(self.streams, newStream(stream))

		case av.AAC:
			aac := stream.(aacparser.CodecData)
			audiodata := self.makeAACAudiodata(aac, flvio.AAC_SEQHDR, aac.MPEG4AudioConfigBytes())
			w := self.writeAudioDataStart()
			audiodata.Marshal(w)
			if err = self.writeAudioDataEnd(0); err != nil {
				return
			}
			self.streams = append(self.streams, newStream(stream))
		}
	}

	self.stage++
	return
}

func (self *Conn) makeH264Videodata(pkttype uint8, iskeyframe bool, data []byte) flvio.Videodata {
	videodata := flvio.Videodata{
		CodecID: flvio.VIDEO_H264,
		AVCPacketType: pkttype,
		Data: data,
	}
	if iskeyframe {
		videodata.FrameType = flvio.FRAME_KEY
	} else {
		videodata.FrameType = flvio.FRAME_INTER
	}
	return videodata
}

func (self *Conn) makeAACAudiodata(stream av.AudioCodecData, pkttype uint8, data []byte) flvio.Audiodata {
	tag := flvio.MakeAACAudiodata(stream, data)
	tag.AACPacketType = pkttype
	return tag
}

func (self *Conn) writeSetChunkSize(size uint32) (err error) {
	w := self.writeProtoCtrlMsgStart()
	w.WriteU32BE(size)
	return self.writeProtoCtrlMsgEnd(msgtypeidSetChunkSize)
}

func (self *Conn) writeWindowAckSize(size uint32) (err error) {
	w := self.writeProtoCtrlMsgStart()
	w.WriteU32BE(size)
	return self.writeProtoCtrlMsgEnd(msgtypeidWindowAckSize)
}

func (self *Conn) writeSetPeerBandwidth(acksize uint32, limittype uint8) (err error) {
	w := self.writeProtoCtrlMsgStart()
	w.WriteU32BE(acksize)
	w.WriteU8(limittype)
	return self.writeProtoCtrlMsgEnd(msgtypeidSetPeerBandwidth)
}

func (self *Conn) writeProtoCtrlMsgStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeProtoCtrlMsgEnd(msgtypeid uint8) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(2, 0, msgtypeid, 0, msgdatav)
}

func (self *Conn) writeCommandMsgStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeCommandMsgEnd(csid uint32, msgsid uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(csid, 0, msgtypeidCommandMsgAMF0, msgsid, msgdatav)
}

func (self *Conn) writeDataMsgStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeDataMsgEnd(csid uint32, msgsid uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(csid, 0, msgtypeidDataMsgAMF0, msgsid, msgdatav)
}

func (self *Conn) writeVideoDataStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeVideoDataEnd(timestamp uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(7, timestamp, msgtypeidVideoMsg, self.avmsgsid, msgdatav)
}

func (self *Conn) writeAudioDataStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeAudioDataEnd(timestamp uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(6, timestamp, msgtypeidAudioMsg, self.avmsgsid, msgdatav)
}

func (self *Conn) writeUserControlMsgStart(eventtype uint16) *pio.Writer {
	self.intw.SaveToVecOn()
	self.intw.WriteU16BE(eventtype)
	return self.intw
}

func (self *Conn) writeUserControlMsgEnd() (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(2, 0, msgtypeidUserControl, 0, msgdatav)
}

func (self *Conn) writeStreamBegin(msgsid uint32) (err error) {
	w := self.writeUserControlMsgStart(eventtypeStreamBegin)
	w.WriteU32BE(msgsid)
	return self.writeUserControlMsgEnd()
}

func (self *Conn) writeSetBufferLength(msgsid uint32, timestamp uint32) (err error) {
	w := self.writeUserControlMsgStart(eventtypeSetBufferLength)
	w.WriteU32BE(msgsid)
	w.WriteU32BE(timestamp)
	return self.writeUserControlMsgEnd()
}

func (self *Conn) writeChunks(csid uint32, timestamp uint32, msgtypeid uint8, msgsid uint32, msgdatav [][]byte) (err error) {
	msgdatalen := pio.VecLen(msgdatav)
	msghdrtype := 0
	var tsdelta uint32

	if false { // always msghdrtype==1 is ok
		cs := self.writecsmap[csid]
		if cs == nil {
			cs = &chunkStream{}
			self.writecsmap[csid] = cs
		} else {
			if msgsid == cs.msgsid {
				if uint32(msgdatalen) == cs.msgdatalen && msgtypeid == cs.msgtypeid {
					if timestamp == cs.timenow {
						msghdrtype = 3
					} else {
						msghdrtype = 2
					}
				} else {
					msghdrtype = 1
				}
			}
			tsdelta = timestamp - cs.timenow
		}
		cs.timenow = timestamp
		cs.msgdatalen = uint32(msgdatalen)
		cs.msgtypeid = msgtypeid
		cs.msgsid = msgsid
	}

	if err = self.bw.WriteU8(byte(csid)&0x3f|byte(msghdrtype)<<6); err != nil {
		return
	}

	switch msghdrtype {
	case 0:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                   timestamp                   |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id| msg stream id |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |           message stream id (cont)            |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 9 Chunk Message Header – Type 0
		if err = self.bw.WriteU24BE(timestamp); err != nil {
			return
		}
		if err = self.bw.WriteU24BE(uint32(msgdatalen)); err != nil {
			return
		}
		if err = self.bw.WriteU8(msgtypeid); err != nil {
			return
		}
		if err = self.bw.WriteU32LE(msgsid); err != nil {
			return
		}

	case 1:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id|
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 10 Chunk Message Header – Type 1
		if err = self.bw.WriteU24BE(tsdelta); err != nil {
			return
		}
		if err = self.bw.WriteU24BE(uint32(msgdatalen)); err != nil {
			return
		}
		if err = self.bw.WriteU8(msgtypeid); err != nil {
			return
		}

	case 2:
		//  0                   1                   2
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 11 Chunk Message Header – Type 2
		if err = self.bw.WriteU24BE(tsdelta); err != nil {
			return
		}

	case 3:
	}

	msgdataoff := 0
	for {
		size := msgdatalen - msgdataoff
		if size > self.writeMaxChunkSize {
			size = self.writeMaxChunkSize
		}

		write := pio.VecSlice(msgdatav, msgdataoff, msgdataoff+size)
		for _, b := range write {
			if _, err = self.bw.Write(b); err != nil {
				return
			}
		}

		msgdataoff += size
		if msgdataoff == msgdatalen {
			break
		}

		// Type 3
		if err = self.bw.WriteU8(byte(csid)&0x3f|3<<6); err != nil {
			return
		}
	}

	if self.Debug {
		fmt.Printf("rtmp: write chunk msgdatalen=%d msgsid=%d\n", msgdatalen, msgsid)
		b := []byte{}
		for _, a := range msgdatav {
			b = append(b, a...)
		}
		fmt.Print(hex.Dump(b))
	}

	if err = self.bufw.Flush(); err != nil {
		return
	}

	return
}

func (self *Conn) readChunk() (err error) {
	var msghdrtype uint8
	var csid uint32
	var header uint8
	if header, err = self.br.ReadU8(); err != nil {
		return
	}
	msghdrtype = header>>6

	csid = uint32(header)&0x3f
	switch csid {
	default: // Chunk basic header 1
	case 0: // Chunk basic header 2
		var i uint8
		if i, err = self.br.ReadU8(); err != nil {
			return
		}
		csid = uint32(i)+64
	case 1: // Chunk basic header 3
		var i uint16
		if i, err = self.br.ReadU16BE(); err != nil {
			return
		}
		csid = uint32(i)+64
	}

	cs := self.readcsmap[csid]
	if cs == nil {
		cs = &chunkStream{}
		self.readcsmap[csid] = cs
	}

	var timestamp uint32

	switch msghdrtype {
	case 0:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                   timestamp                   |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id| msg stream id |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |           message stream id (cont)            |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 9 Chunk Message Header – Type 0
		if cs.msgdataleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgdataleft=%d invalid", cs.msgdataleft)
			return
		}
		var h[]byte
		if h, err = self.br.ReadBytes(11); err != nil {
			return
		}
		timestamp = pio.GetU24BE(h[0:3])
		cs.msghdrtype = msghdrtype
		cs.msgdatalen = pio.GetU24BE(h[3:6])
		cs.msgtypeid = h[6]
		cs.msgsid = pio.GetU32LE(h[7:11])
		if timestamp == 0xffffff {
			if timestamp, err = self.br.ReadU32BE(); err != nil {
				return
			}
			cs.hastimeext = true
		} else {
			cs.hastimeext = false
		}
		cs.timenow = timestamp
		cs.Start()

	case 1:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id|
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 10 Chunk Message Header – Type 1
		if cs.msgdataleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgdataleft=%d invalid", cs.msgdataleft)
			return
		}
		var h[]byte
		if h, err = self.br.ReadBytes(7); err != nil {
			return
		}
		timestamp = pio.GetU24BE(h[0:3])
		cs.msghdrtype = msghdrtype
		cs.msgdatalen = pio.GetU24BE(h[3:6])
		cs.msgtypeid = h[6]
		if timestamp == 0xffffff {
			if timestamp, err = self.br.ReadU32BE(); err != nil {
				return
			}
			cs.hastimeext = true
		} else {
			cs.hastimeext = false
		}
		cs.timedelta = timestamp
		cs.timenow += timestamp
		cs.Start()

	case 2:
		//  0                   1                   2
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 11 Chunk Message Header – Type 2
		if cs.msgdataleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgdataleft=%d invalid", cs.msgdataleft)
			return
		}
		var h[]byte
		if h, err = self.br.ReadBytes(3); err != nil {
			return
		}
		cs.msghdrtype = msghdrtype
		timestamp = pio.GetU24BE(h[0:3])
		if timestamp == 0xffffff {
			if timestamp, err = self.br.ReadU32BE(); err != nil {
				return
			}
			cs.hastimeext = true
		} else {
			cs.hastimeext = false
		}
		cs.timedelta = timestamp
		cs.timenow += timestamp
		cs.Start()

	case 3:
		if cs.msgdataleft == 0 {
			switch cs.msghdrtype {
			case 0:
				if cs.hastimeext {
					if timestamp, err = self.br.ReadU32BE(); err != nil {
						return
					}
					cs.timenow = timestamp
				}
			case 1, 2:
				if cs.hastimeext {
					if timestamp, err = self.br.ReadU32BE(); err != nil {
						return
					}
				} else {
					timestamp = cs.timedelta
				}
				cs.timenow += timestamp
			}
			cs.Start()
		}

	default:
		err = fmt.Errorf("rtmp: invalid chunk msg header type=%d", msghdrtype)
		return
	}

	size := int(cs.msgdataleft)
	if size > self.readMaxChunkSize {
		size = self.readMaxChunkSize
	}
	off := cs.msgdatalen-cs.msgdataleft
	buf := cs.msgdata[off:int(off)+size]
	if _, err = io.ReadFull(self.br, buf); err != nil {
		return
	}
	cs.msgdataleft -= uint32(size)

	if self.Debug {
		fmt.Printf("rtmp: chunk msgsid=%d msgtypeid=%d msghdrtype=%d len=%d left=%d\n",
			cs.msgsid, cs.msgtypeid, cs.msghdrtype, cs.msgdatalen, cs.msgdataleft)
	}

	if cs.msgdataleft == 0 {
		if self.Debug {
			fmt.Println("rtmp: chunk data")
			fmt.Print(hex.Dump(cs.msgdata))
		}

		if err = self.handleMsg(cs.timenow, cs.msgsid, cs.msgtypeid, cs.msgdata); err != nil {
			return
		}
	}

	return
}

func (self *Conn) handleCommandMsgAMF0(r *pio.Reader) (err error) {
	commandname, _ := flvio.ReadAMF0Val(r)
	commandtransid, _ := flvio.ReadAMF0Val(r)
	commandobj, _ := flvio.ReadAMF0Val(r)

	var ok bool
	if self.commandname, ok = commandname.(string); !ok {
		err = fmt.Errorf("rtmp: CommandMsgAMF0 command is not string")
		return
	}

	self.commandobj, _ = commandobj.(flvio.AMFMap)
	self.commandtransid, _ = commandtransid.(float64)
	self.commandparams = []interface{}{}
	for {
		if val, rerr := flvio.ReadAMF0Val(r); rerr != nil {
			break
		} else {
			self.commandparams = append(self.commandparams, val)
		}
	}

	self.gotcommand = true
	return
}

func (self *Conn) handleMsg(timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {
	self.msgtypeid = msgtypeid
	self.timestamp = timestamp

	switch msgtypeid {
	case msgtypeidCommandMsgAMF0:
		r := pio.NewReaderBytes(msgdata)
		if err = self.handleCommandMsgAMF0(r); err != nil {
			return
		}

	case msgtypeidCommandMsgAMF3:
		r := pio.NewReaderBytes(msgdata)
		r.ReadU8() // skip first byte
		if err = self.handleCommandMsgAMF0(r); err != nil {
			return
		}

	case msgtypeidUserControl:
		if len(msgdata) >= 2 {
			self.eventtype = pio.GetU16BE(msgdata)
			self.msgdata = msgdata
		} else {
			err = fmt.Errorf("rtmp: short packet of UserControl")
			return
		}

	case msgtypeidDataMsgAMF0:
		r := pio.NewReaderBytes(msgdata)
		for {
			if val, err := flvio.ReadAMF0Val(r); err != nil {
				break
			} else {
				self.datamsgvals = append(self.datamsgvals, val)
			}
		}

	case msgtypeidVideoMsg:
		tag := &flvio.Videodata{}
		r := pio.NewReaderBytes(msgdata)
		r.LimitOn(int64(len(msgdata)))
		if err = tag.Unmarshal(r); err != nil {
			return
		}
		self.videodata = tag

	case msgtypeidAudioMsg:
		tag := &flvio.Audiodata{}
		r := pio.NewReaderBytes(msgdata)
		r.LimitOn(int64(len(msgdata)))
		if err = tag.Unmarshal(r); err != nil {
			return
		}
		self.audiodata = tag

	case msgtypeidSetChunkSize:
		self.readMaxChunkSize = int(pio.GetU32BE(msgdata))
		return
	}

	self.gotmsg = true
	return
}

var (
	hsClientFullKey = []byte{
		'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
		'F', 'l', 'a', 's', 'h', ' ', 'P', 'l', 'a', 'y', 'e', 'r', ' ',
		'0', '0', '1',
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
		0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	hsServerFullKey = []byte{
		'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
		'F', 'l', 'a', 's', 'h', ' ', 'M', 'e', 'd', 'i', 'a', ' ',
		'S', 'e', 'r', 'v', 'e', 'r', ' ',
		'0', '0', '1',
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
		0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	hsClientPartialKey = hsClientFullKey[:30]
	hsServerPartialKey = hsServerFullKey[:36]
)

func hsMakeDigest(key []byte, src []byte, gap int) (dst []byte) {
	h := hmac.New(sha256.New, key)
	if gap <= 0 {
		h.Write(src)
	} else {
		h.Write(src[:gap])
		h.Write(src[gap+32:])
	}
	return h.Sum(nil)
}

func hsCalcDigestPos(p []byte, base int) (pos int) {
	for i := 0; i < 4; i++ {
		pos += int(p[base+i])
	}
	pos = (pos%728)+base+4
	return
}

func hsFindDigest(p []byte, key []byte, base int) int {
	gap := hsCalcDigestPos(p, base)
	digest := hsMakeDigest(key, p, gap)
	if bytes.Compare(p[gap:gap+32], digest) != 0 {
		return -1
	}
	return gap
}

func hsParse1(p []byte, peerkey []byte, key []byte) (ok bool, digest []byte) {
	var pos int
	if pos = hsFindDigest(p, peerkey, 772); pos == -1 {
		if pos = hsFindDigest(p, peerkey, 8); pos == -1 {
			return
		}
	}
	ok = true
	digest = hsMakeDigest(key, p[pos:pos+32], -1)
	return
}

func hsCreate01(p []byte, time uint32, ver uint32, key []byte) {
	p[0] = 3
	p1 := p[1:]
	rand.Read(p1[8:])
	pio.PutU32BE(p1[0:4], time)
	pio.PutU32BE(p1[4:8], ver)
	gap := hsCalcDigestPos(p1, 8)
	digest := hsMakeDigest(key, p1, gap)
	copy(p1[gap:], digest)
}

func hsCreate2(p []byte, key []byte) {
	rand.Read(p)
	gap := len(p)-32
	digest := hsMakeDigest(key, p, gap)
	copy(p[gap:], digest)
}

func (self *Conn) handshakeClient() (err error) {
	var random [(1+1536*2)*2]byte

	C0C1C2 := random[:1536*2+1]
	C0 := C0C1C2[:1]
	//C1 := C0C1C2[1:1536+1]
	C0C1 := C0C1C2[:1536+1]
	C2 := C0C1C2[1536+1:]

	S0S1S2 := random[1536*2+1:]
	//S0 := S0S1S2[:1]
	S1 := S0S1S2[1:1536+1]
	//S0S1 := S0S1S2[:1536+1]
	//S2 := S0S1S2[1536+1:]

	C0[0] = 3
	//hsCreate01(C0C1, hsClientFullKey)

	// > C0C1
	if _, err = self.bw.Write(C0C1); err != nil {
		return
	}
	if err = self.bufw.Flush(); err != nil {
		return
	}

	// < S0S1S2
	if _, err = io.ReadFull(self.br, S0S1S2); err != nil {
		return
	}

	if self.Debug {
		fmt.Println("rtmp: handshakeClient: server version", S1[4],S1[5],S1[6],S1[7])
	}

	if ver := pio.GetU32BE(S1[4:8]); ver != 0 {
		C2 = S1
	} else {
		C2 = S1
	}

	// > C2
	if _, err = self.bw.Write(C2); err != nil {
		return
	}
	if err = self.bufw.Flush(); err != nil {
		return
	}

	self.stage++
	return
}

func (self *Conn) handshakeServer() (err error) {
	var random [(1+1536*2)*2]byte

	C0C1C2 := random[:1536*2+1]
	C0 := C0C1C2[:1]
	C1 := C0C1C2[1:1536+1]
	C0C1 := C0C1C2[:1536+1]
	C2 := C0C1C2[1536+1:]

	S0S1S2 := random[1536*2+1:]
	//S0 := S0S1S2[:1]
	S1 := S0S1S2[1:1536+1]
	S0S1 := S0S1S2[:1536+1]
	S2 := S0S1S2[1536+1:]

	// < C0C1
	if _, err = io.ReadFull(self.br, C0C1); err != nil {
		return
	}
	if C0[0] != 3 {
		err = fmt.Errorf("rtmp: handshake version=%d invalid", C0[0])
		return
	}

	clitime := pio.GetU32BE(C1[0:4])
	srvtime := clitime
	srvver := uint32(0x0d0e0a0d)
	cliver := pio.GetU32BE(C1[4:8])

	if cliver != 0 {
		var ok bool
		var digest []byte
		if ok, digest = hsParse1(C1, hsClientPartialKey, hsServerFullKey); !ok {
			err = fmt.Errorf("rtmp: handshake server: C1 invalid")
			return
		}
		hsCreate01(S0S1, srvtime, srvver, hsServerPartialKey)
		hsCreate2(S2, digest)
	} else {
		copy(S1, C1)
		copy(S2, C2)
	}

	// > S0S1S2
	if _, err = self.bw.Write(S0S1S2); err != nil {
		return
	}
	if err = self.bufw.Flush(); err != nil {
		return
	}

	// < C2
	if _, err = io.ReadFull(self.br, C2); err != nil {
		return
	}

	self.stage++
	return
}

func Handler(h *avutil.RegisterHandler) {
	h.UrlDemuxer = func(uri string) (ok bool, demuxer av.DemuxCloser, err error) {
		if !strings.HasPrefix(uri, "rtmp://") {
			return
		}
		ok = true
		demuxer, err = Dial(uri)
		return
	}
}

