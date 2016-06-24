
package rtmp

import (
	"bytes"
	"net"
	"bufio"
	"time"
	"fmt"
	"encoding/hex"
	"io"
	"github.com/nareix/pio"
	"github.com/nareix/flv/flvio"
	"github.com/nareix/av"
	"github.com/nareix/codec/h264parser"
	"github.com/nareix/codec/aacparser"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/rand"
)

type Server struct {
	Addr string
	HandlePublish func(*Conn)
	HandlePlay func(*Conn)
}

func (self *Server) handleConn(conn *Conn) (err error) {
	if err = conn.determineType(); err != nil {
		fmt.Println("rtmp: conn closed:", err)
		return
	}

	if conn.playing {
		if self.HandlePlay != nil {
			self.HandlePlay(conn)
			conn.Close()
		}
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

	var netconn net.Conn
	for {
		if netconn, err = listener.Accept(); err != nil {
			return
		}

		conn := newConn(netconn)
		go self.handleConn(conn)
	}
}

type Conn struct {
	RequestUri string
	streams []av.CodecData

	br *pio.Reader
	bw *pio.Writer
	bufr *bufio.Reader
	bufw *bufio.Writer
	intw *pio.Writer
	netconn net.Conn

	writeMaxChunkSize int
	readMaxChunkSize int

	lastcsid uint32
	lastcs *chunkStream
	csmap map[uint32]*chunkStream

	publishing, playing bool
	playmsgcsid uint32

	gotcommand bool
	commandname string
	commandtransid float64
	commandobj flvio.AMFMap
	commandparams []interface{}

	gotmsg bool
	msgdata []byte
	msgtypeid uint8
	msgcsid uint32

	eventtype uint16
}

func newConn(netconn net.Conn) *Conn {
	conn := &Conn{}
	conn.netconn = netconn
	conn.csmap = make(map[uint32]*chunkStream)
	conn.readMaxChunkSize = 128
	conn.writeMaxChunkSize = 128
	conn.bufr = bufio.NewReaderSize(netconn, 512)
	conn.bufw = bufio.NewWriterSize(netconn, 512)
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
	for {
		if err = self.readChunk(); err != nil {
			return
		}
		if self.gotmsg {
			fmt.Println("rtmp: gotmsg iscommand", self.gotcommand)
			return
		}
	}
}

func (self *Conn) determineType() (err error) {
	if err = self.handshake(); err != nil {
		return
	}

	// < connect
	if err = self.pollCommand(); err != nil {
		return
	}
	if self.commandname != "connect" {
		err = fmt.Errorf("rtmp: first command is not connect")
		return
	}

	// > WindowAckSize
	if err = self.writeWindowAckSize(5000000); err != nil {
		return
	}
	// > SetPeerBandwidth
	if err = self.writeSetPeerBandwidth(5000000, 2); err != nil {
		return
	}
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
		"description": "Connection Success.",
		"objectEncoding": 3,
	})
	self.writeCommandMsgEnd(3, 0)

	for {
		if err = self.pollMsg(); err != nil {
			return
		}
		if self.gotcommand {
			switch self.commandname {

			// < createStream
			case "createStream":
				self.playmsgcsid = uint32(1)
				// > _result(streamid)
				w := self.writeCommandMsgStart()
				flvio.WriteAMF0Val(w, "_result")
				flvio.WriteAMF0Val(w, self.commandtransid)
				flvio.WriteAMF0Val(w, nil)
				flvio.WriteAMF0Val(w, self.playmsgcsid) // streamid=1
				self.writeCommandMsgEnd(3, 0)

			// < play("path")
			case "play":
				if len(self.commandparams) < 1 {
					err = fmt.Errorf("rtmp: play params invalid")
					return
				}
				path, _ := self.commandparams[0].(string)
				self.RequestUri = path
				fmt.Println("rtmp: play", path)

				// > streamBegin(streamid)
				self.writeStreamBegin(self.playmsgcsid)

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
				self.writeCommandMsgEnd(5, self.playmsgcsid)

				// > |RtmpSampleAccess()
				w = self.writeDataMsgStart()
				flvio.WriteAMF0Val(w, "|RtmpSampleAccess")
				flvio.WriteAMF0Val(w, true)
				flvio.WriteAMF0Val(w, true)
				self.writeDataMsgEnd(5, self.playmsgcsid)

				fmt.Println("rtmp: playing")
				self.playing = true
				return
			}

		}
	}

	return
}

func (self *Conn) WritePacket(pkt av.Packet) (err error) {
	ts := uint32(pkt.Time/time.Millisecond)
	stream := self.streams[pkt.Idx]

	switch stream.Type() {
	case av.AAC:
		audiodata := self.makeAACAudiodata(stream.(av.AudioCodecData), flvio.AAC_RAW, pkt.Data)
		w := self.writeAudioDataStart()
		audiodata.Marshal(w)
		self.writeAudioDataEnd(ts)

	case av.H264:
		videodata := self.makeH264Videodata(flvio.AVC_NALU, pkt.Data)
		w := self.writeVideoDataStart()
		videodata.Marshal(w)
		self.writeVideoDataEnd(ts)
	}
	return
}

func (self *Conn) WriteHeader(streams []av.CodecData) (err error) {
	metadata := flvio.AMFMap{}
	metadata["Server"] = "joy4"
	metadata["duration"] = 0

	for _, _stream := range streams {
		typ := _stream.Type()
		switch {
		case typ.IsVideo():
			stream := _stream.(av.VideoCodecData)
			switch typ {
			case av.H264:
				metadata["videocodecid"] = flvio.VIDEO_H264

			default:
				err = fmt.Errorf("rtmp: WriteHeader unsupported video codecType=%v", stream.Type())
				return
			}

			metadata["width"] = stream.Width()
			metadata["height"] = stream.Height()
			metadata["displayWidth"] = stream.Width()
			metadata["displayHeight"] = stream.Height()
			metadata["framerate"] = 24 // TODO: make it correct
			metadata["fps"] = 24
			metadata["videodatarate"] = 1538 // TODO: make it correct
			metadata["profile"] = ""
			metadata["level"] = ""

		case typ.IsAudio():
			stream := _stream.(av.AudioCodecData)
			switch typ {
			case av.AAC:
				metadata["audiocodecid"] = flvio.SOUND_AAC

			default:
				err = fmt.Errorf("rtmp: WriteHeader unsupported audio codecType=%v", stream.Type())
				return
			}

			metadata["audiodatarate"] = 156 // TODO: make it correct
		}
	}

	// > onMetaData()
	w := self.writeDataMsgStart()
	flvio.WriteAMF0Val(w, "onMetaData")
	flvio.WriteAMF0Val(w, metadata)
	if err = self.writeDataMsgEnd(5, self.playmsgcsid); err != nil {
		return
	}

	// > Videodata(decoder config)
	// > Audiodata(decoder config)
	for _, stream := range streams {
		switch stream.Type() {
		case av.H264:
			h264 := stream.(h264parser.CodecData)
			videodata := self.makeH264Videodata(flvio.AVC_SEQHDR, h264.AVCDecoderConfRecordBytes())
			w := self.writeVideoDataStart()
			videodata.Marshal(w)
			if err = self.writeVideoDataEnd(0); err != nil {
				return
			}

		case av.AAC:
			aac := stream.(aacparser.CodecData)
			audiodata := self.makeAACAudiodata(aac, flvio.AAC_SEQHDR, aac.MPEG4AudioConfigBytes())
			w := self.writeAudioDataStart()
			audiodata.Marshal(w)
			if err = self.writeAudioDataEnd(0); err != nil {
				return
			}
		}
	}

	self.streams = streams
	return
}

func (self *Conn) makeH264Videodata(pkttype uint8, data []byte) flvio.Videodata {
	return flvio.Videodata{
		FrameType: flvio.FRAME_KEY,
		CodecID: flvio.VIDEO_H264,
		AVCPacketType: pkttype,
		Data: data,
	}
}

func (self *Conn) makeAACAudiodata(stream av.AudioCodecData, pkttype uint8, data []byte) flvio.Audiodata {
	audiodata := flvio.Audiodata{
		SoundFormat: flvio.SOUND_AAC,
		SoundRate: flvio.SOUND_44Khz,
		AACPacketType: pkttype,
	}
	switch stream.SampleFormat().BytesPerSample() {
	case 1:
		audiodata.SoundSize = flvio.SOUND_8BIT
	case 2:
		audiodata.SoundSize = flvio.SOUND_16BIT
	}
	switch stream.ChannelLayout().Count() {
	case 1:
		audiodata.SoundType = flvio.SOUND_MONO
	case 2:
		audiodata.SoundType = flvio.SOUND_STEREO
	}
	return audiodata
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

func (self *Conn) writeCommandMsgEnd(csid uint32, msgcsid uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(csid, 0, msgtypeidCommandMsgAMF0, msgcsid, msgdatav)
}

func (self *Conn) writeDataMsgStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeDataMsgEnd(csid uint32, msgcsid uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(csid, 0, msgtypeidDataMsgAMF0, msgcsid, msgdatav)
}

func (self *Conn) writeVideoDataStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeVideoDataEnd(timestamp uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(6, timestamp, msgtypeidVideoMsg, self.playmsgcsid, msgdatav)
}

func (self *Conn) writeAudioDataStart() *pio.Writer {
	self.intw.SaveToVecOn()
	return self.intw
}

func (self *Conn) writeAudioDataEnd(timestamp uint32) (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(6, timestamp, msgtypeidAudioMsg, self.playmsgcsid, msgdatav)
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

func (self *Conn) writeStreamBegin(msgcsid uint32) (err error) {
	w := self.writeUserControlMsgStart(eventtypeStreamBegin)
	w.WriteU32BE(msgcsid)
	return self.writeUserControlMsgEnd()
}

func (self *Conn) writeChunks(csid uint32, timestamp uint32, msgtypeid uint8, msgcsid uint32, msgdatav [][]byte) (err error) {
	msgdatalen := pio.VecLen(msgdatav)

	// [Type 0][Type 3][Type 3]....

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
	if err = self.bw.WriteU8(byte(csid)&0x3f); err != nil {
		return
	}
	if err = self.bw.WriteU24BE(timestamp); err != nil {
		return
	}
	if err = self.bw.WriteU24BE(uint32(msgdatalen)); err != nil {
		return
	}
	if err = self.bw.WriteU8(msgtypeid); err != nil {
		return
	}
	if err = self.bw.WriteU32BE(msgcsid); err != nil {
		return
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

	fmt.Printf("rtmp: write chunk msgdatalen=%d\n", msgdatalen)

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

	var cs *chunkStream
	if self.lastcs != nil && self.lastcsid == csid {
		cs = self.lastcs
	} else {
		cs = &chunkStream{}
		self.csmap[csid] = cs
	}
	self.lastcs = cs
	self.lastcsid = csid

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
		cs.msgsid = pio.GetU32BE(h[7:11])
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

	if true {
		fmt.Printf("rtmp: chunk csid=%d msgsid=%d msgtypeid=%d msghdrtype=%d len=%d left=%d\n",
			csid, cs.msgsid, cs.msgtypeid, cs.msghdrtype, cs.msgdatalen, cs.msgdataleft)
	}

	if cs.msgdataleft == 0 {
		if true {
			fmt.Println("rtmp: chunk data")
			fmt.Print(hex.Dump(cs.msgdata))
			fmt.Printf("%x\n", cs.msgdata)
		}

		if err = self.handleMsg(csid, cs.msgtypeid, cs.msgdata); err != nil {
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

func (self *Conn) handleMsg(msgcsid uint32, msgtypeid uint8, msgdata []byte) (err error) {
	self.msgcsid = msgcsid

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
		} else {
			err = fmt.Errorf("rtmp: short packet of UserControl")
			return
		}

	case msgtypeidSetPeerBandwidth:
	case msgtypeidSetChunkSize:
	case msgtypeidWindowAckSize:
		self.msgdata = msgdata
		self.msgtypeid = msgtypeid

	default:
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

func hsMakeDigest(key []byte, src []byte, skip int) (dst []byte) {
	h := hmac.New(sha256.New, key)
	if skip >= 0 && skip < len(src) {
		if skip != 0 {
			h.Write(src[:skip])
		}
		if len(src) != skip + 32 {
			h.Write(src[skip+32:])
		}
	} else {
		h.Write(src)
	}
	return h.Sum(nil)
}

func hsFindDigest(p []byte, key []byte, base int) (off int) {
	for n := 0; n < 4; n++ {
		off += int(p[base + n])
	}
	off = (off % 728) + base + 4
	digest := hsMakeDigest(key, p, off)
	if bytes.Compare(p[off:off+32], digest) != 0 {
		off = -1
	}
	return off
}

func hsParseC1(p []byte) (digest []byte, err error) {
	return hsParse1(p, hsClientPartialKey)
}

func hsParseS1(p []byte) (digest []byte, err error) {
	return hsParse1(p, hsServerPartialKey)
}

func hsParse1(p []byte, key []byte) (digest []byte, err error) {
	var off int
	if off = hsFindDigest(p, key, 772); off == -1 {
		if off = hsFindDigest(p, key, 8); off == -1 {
			err = fmt.Errorf("rtmp: handshake: C1 parse failed")
			return
		}
	}
	digest = hsMakeDigest(key, p[off:off+32], -1)
	return
}

func hsCreateS1(p []byte) {
	hsCreate1(p, hsServerPartialKey)
}

func hsCreateS2(p []byte, digest []byte) {
	rand.Read(p)
	digest2 := hsMakeDigest(digest, p, 1536-32)
	copy(p[1536-32:], digest2)
}

func hsCreate1(p []byte, key []byte) {
	rand.Read(p)
	off := 0
	for n := 8; n < 12; n++ {
		off += int(p[n])
	}
	off = (off % 728) + 12
	digest := hsMakeDigest(key, p, off)
	copy(p[off:], digest)
}

func (self *Conn) handshake() (err error) {
	var version uint8

	var random [1536*4]byte
	var digest []byte
	C1 := random[0:1536]
	S1 := random[1536:1536*2]
	C2 := random[1536*2:1536*3]
	S2 := random[1536*3:1536*4]

	// C0
	if version, err = self.br.ReadU8(); err != nil {
		return
	}
	if version != 0x3 {
		err = fmt.Errorf("rtmp: handshake C0: version=%d invalid", version)
		return
	}
	// C1
	if _, err = io.ReadFull(self.br, C1); err != nil {
		return
	}

	// TODO: do the right thing
	if false {
		if digest, err = hsParseC1(C1); err != nil {
			return
		}
		serverTime := uint32(0)
		serverVer := uint32(0x0d0e0a0d)
		hsCreateS1(S1)
		pio.PutU32BE(S1[0:4], serverTime)
		pio.PutU32BE(S1[4:8], serverVer)
		hsCreateS2(S2, digest)
	}

	// S0
	if err = self.bw.WriteU8(0x3); err != nil {
		return
	}
	// S1
	if _, err = self.bw.Write(S1); err != nil {
		return
	}
	// S2
	if _, err = self.bw.Write(S2); err != nil {
		return
	}
	if err = self.bufw.Flush(); err != nil {
		return
	}

	// C2
	if _, err = io.ReadFull(self.br, C2); err != nil {
		return
	}

	return
}

