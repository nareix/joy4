
package rtmp

import (
	"net"
	"bufio"
	"fmt"
	"encoding/hex"
	"io"
	"github.com/nareix/pio"
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
	if err = conn.Handshake(); err != nil {
		return
	}

	for {
		if err = conn.ReadChunk(); err != nil {
			return
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

		conn := &Conn{}
		conn.csmap = make(map[uint32]*chunkStream)
		conn.maxChunkSize = 128
		conn.bufr = bufio.NewReaderSize(netconn, 512)
		conn.bufw = bufio.NewWriterSize(netconn, 512)
		conn.br = pio.NewReader(conn.bufr)
		conn.bw = pio.NewWriter(conn.bufw)
		go self.handleConn(conn)
	}
}

type Conn struct {
	br *pio.Reader
	bw *pio.Writer
	bufr *bufio.Reader
	bufw *bufio.Writer

	maxChunkSize int

	lastcsid uint32
	lastcs *chunkStream
	csmap map[uint32]*chunkStream
}

type chunkStream struct {
	TimestampNow uint32
	TimestampDelta uint32
	HasTimestampExt bool
	Msgsid uint32
	Msgtypeid uint8
	Msglen uint32
	Msgleft uint32
	Msghdrtype uint8
	Msgdata []byte
}

func (self *chunkStream) Start() {
	self.Msgleft = self.Msglen
	self.Msgdata = make([]byte, self.Msglen)
}

func (self *Conn) ReadChunk() ( err error) {
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
		if cs.Msgleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgleft=%d invalid", cs.Msgleft)
			return
		}
		var h[]byte
		if h, err = self.br.ReadBytes(11); err != nil {
			return
		}
		timestamp = pio.GetU24BE(h[0:3])
		cs.Msghdrtype = msghdrtype
		cs.Msglen = pio.GetU24BE(h[3:6])
		cs.Msgtypeid = h[6]
		cs.Msgsid = pio.GetU32BE(h[7:11])
		if timestamp == 0xffffff {
			if timestamp, err = self.br.ReadU32BE(); err != nil {
				return
			}
			cs.HasTimestampExt = true
		} else {
			cs.HasTimestampExt = false
		}
		cs.TimestampNow = timestamp
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
		if cs.Msgleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgleft=%d invalid", cs.Msgleft)
			return
		}
		var h[]byte
		if h, err = self.br.ReadBytes(7); err != nil {
			return
		}
		timestamp = pio.GetU24BE(h[0:3])
		cs.Msghdrtype = msghdrtype
		cs.Msglen = pio.GetU24BE(h[3:6])
		cs.Msgtypeid = h[6]
		if timestamp == 0xffffff {
			if timestamp, err = self.br.ReadU32BE(); err != nil {
				return
			}
			cs.HasTimestampExt = true
		} else {
			cs.HasTimestampExt = false
		}
		cs.TimestampDelta = timestamp
		cs.TimestampNow += timestamp
		cs.Start()

	case 2:
		//  0                   1                   2
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 11 Chunk Message Header – Type 2
		if cs.Msgleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgleft=%d invalid", cs.Msgleft)
			return
		}
		var h[]byte
		if h, err = self.br.ReadBytes(3); err != nil {
			return
		}
		cs.Msghdrtype = msghdrtype
		timestamp = pio.GetU24BE(h[0:3])
		if timestamp == 0xffffff {
			if timestamp, err = self.br.ReadU32BE(); err != nil {
				return
			}
			cs.HasTimestampExt = true
		} else {
			cs.HasTimestampExt = false
		}
		cs.TimestampDelta = timestamp
		cs.TimestampNow += timestamp
		cs.Start()

	case 3:
		if cs.Msgleft == 0 {
			switch cs.Msghdrtype {
			case 0:
				if cs.HasTimestampExt {
					if timestamp, err = self.br.ReadU32BE(); err != nil {
						return
					}
					cs.TimestampNow = timestamp
				}
			case 1, 2:
				if cs.HasTimestampExt {
					if timestamp, err = self.br.ReadU32BE(); err != nil {
						return
					}
				} else {
					timestamp = cs.TimestampDelta
				}
				cs.TimestampNow += timestamp
			}
			cs.Start()
		}

	default:
		err = fmt.Errorf("rtmp: invalid chunk msg header type=%d", msghdrtype)
		return
	}

	size := int(cs.Msgleft)
	if size > self.maxChunkSize {
		size = self.maxChunkSize
	}
	off := cs.Msglen-cs.Msgleft
	buf := cs.Msgdata[off:int(off)+size]
	if _, err = io.ReadFull(self.br, buf); err != nil {
		return
	}
	cs.Msgleft -= uint32(size)

	if true {
		fmt.Printf("rtmp: chunk csid=%d msgsid=%d msgtypeid=%d msghdrtype=%d len=%d left=%d\n",
			csid, cs.Msgsid, cs.Msgtypeid, cs.Msghdrtype, cs.Msglen, cs.Msgleft)
	}

	if cs.Msgleft == 0 {
		if true {
			fmt.Println("rtmp: chunk data")
			fmt.Print(hex.Dump(cs.Msgdata))
			fmt.Printf("%x\n", cs.Msgdata)
		}
	}

	return
}

func (self *Conn) Handshake() (err error) {
	// C0
	var version uint8
	if version, err = self.br.ReadU8(); err != nil {
		return
	}
	if version != 0x3 {
		err = fmt.Errorf("rtmp: handshake c0: version=%d invalid", version)
		return
	}

	// S0
	if err = self.bw.WriteU8(0x3); err != nil {
		return
	}

	random := make([]byte, 1528)

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

	// C1
	var time uint32
	if time, err = self.br.ReadU32BE(); err != nil {
		return
	}
	if _, err = self.br.ReadU32BE(); err != nil {
		return
	}
	if _, err = io.ReadFull(self.br, random); err != nil {
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

