
package rtmp

import (
	"net"
	"bufio"
	"fmt"
	"encoding/hex"
	"io"
	"github.com/nareix/pio"
	"github.com/nareix/flv/flvio"
)

type Publisher struct {
}

type Player struct {
}

type Server struct {
	Addr string
	HandlePublish func(*Publisher)
	HandlePlay func(*Player)
}

func (self *Server) handleConn(conn *Conn) (err error) {
	if err = conn.determineType(); err != nil {
		fmt.Println("rtmp: conn closed:", err)
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
	br *pio.Reader
	bw *pio.Writer
	bufr *bufio.Reader
	bufw *bufio.Writer
	intw *pio.Writer

	writeMaxChunkSize int
	readMaxChunkSize int

	lastcsid uint32
	lastcs *chunkStream
	csmap map[uint32]*chunkStream

	publishing, playing bool

	gotcommand bool
	command string
	commandr *pio.Reader
	commandobj flvio.AMFMap
	commandtransid float64

	gotmsg bool
	msgdata []byte
	msgtypeid uint8
}

func newConn(netconn net.Conn) *Conn {
	conn := &Conn{}
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
)

const (
	eventtypeStreamBegin = 0
)

func (self *Conn) pollCommand() (err error) {
	for {
		if err = self.readChunk(); err != nil {
			return
		}
		if self.gotcommand {
			self.gotcommand = false
			return
		}
	}
}

func (self *Conn) pollMsg() (err error) {
	for {
		if err = self.readChunk(); err != nil {
			return
		}
		if self.gotmsg {
			self.gotmsg = false
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
	if self.command != "connect" {
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
		"objectEncoding": 0,
	})
	self.writeCommandMsgEnd()

	if err = self.pollCommand(); err != nil {
		return
	}
	if err = self.pollCommand(); err != nil {
		return
	}

	return
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

func (self *Conn) writeCommandMsgEnd() (err error) {
	msgdatav := self.intw.SaveToVecOff()
	return self.writeChunks(3, 0, msgtypeidCommandMsgAMF0, 0, msgdatav)
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

	// [Type 0][Type 3][Type 3][Type 3]

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

		if err = self.handleMsg(cs.msgtypeid, cs.msgdata); err != nil {
			return
		}
	}

	return
}

func (self *Conn) handleMsg(msgtypeid uint8, msgdata []byte) (err error) {
	switch msgtypeid {
	case msgtypeidCommandMsgAMF0:
		r := pio.NewReaderBytes(msgdata)

		command, _ := flvio.ReadAMF0Val(r)
		commandtransid, _ := flvio.ReadAMF0Val(r)
		commandobj, _ := flvio.ReadAMF0Val(r)

		var ok bool
		if self.command, ok = command.(string); !ok {
			err = fmt.Errorf("rtmp: CommandMsgAMF0 command is not string")
			return
		}

		self.commandobj, _ = commandobj.(flvio.AMFMap)
		self.commandtransid, _ = commandtransid.(float64)

		self.commandr = r
		self.gotcommand = true

	case msgtypeidSetPeerBandwidth:
	case msgtypeidSetChunkSize:
	case msgtypeidWindowAckSize:
		self.msgdata = msgdata
		self.msgtypeid = msgtypeid
		self.gotmsg = true
	}

	return
}

func (self *Conn) handshake() (err error) {
	var time uint32
	var version uint8
	random := make([]byte, 1528)

	// C0
	if version, err = self.br.ReadU8(); err != nil {
		return
	}
	if version != 0x3 {
		err = fmt.Errorf("rtmp: handshake c0: version=%d invalid", version)
		return
	}
	// C1
	if time, err = self.br.ReadU32BE(); err != nil {
		return
	}
	if _, err = self.br.ReadU32BE(); err != nil {
		return
	}
	if _, err = io.ReadFull(self.br, random); err != nil {
		return
	}

	// S0
	if err = self.bw.WriteU8(0x3); err != nil {
		return
	}
	// S1
	if err = self.bw.WriteU32BE(0); err != nil {
		return
	}
	if err = self.bw.WriteU32BE(0); err != nil {
		return
	}
	if _, err = self.bw.Write(random); err != nil {
		return
	}
	if err = self.bufw.Flush(); err != nil {
		return
	}
	// S2
	if err = self.bw.WriteU32BE(0); err != nil {
		return
	}
	if err = self.bw.WriteU32BE(time); err != nil {
		return
	}
	if _, err = self.bw.Write(random); err != nil {
		return
	}
	if err = self.bufw.Flush(); err != nil {
		return
	}

	// C2
	if time, err = self.br.ReadU32BE(); err != nil {
		return
	}
	if _, err = self.br.ReadU32BE(); err != nil {
		return
	}
	if _, err = io.ReadFull(self.br, random); err != nil {
		return
	}

	return
}

