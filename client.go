package rtsp

import (
	"fmt"
	"net"
	"bytes"
	"io"
	"strings"
	"strconv"
	"bufio"
	"html"
	"net/textproto"
	"net/url"
	"encoding/hex"
	"encoding/binary"
	"encoding/base64"
	"crypto/md5"
	"github.com/nareix/av"
	"github.com/nareix/codec/h264parser"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/codec"
	"github.com/nareix/rtsp/sdp"
	"github.com/nareix/av/pktqueue"
)

type Client struct {
	DebugConn bool
	Headers []string
	url *url.URL
	conn net.Conn
	rconn io.Reader
	requestUri string
	cseq uint
	streams []*Stream
	session string
	authorization []string
	body io.Reader
	pktque *pktqueue.Queue
}

type Request struct {
	Header []string
	Uri string
	Method string
}

type Response struct {
	BlockLength int
	Block []byte
	BlockNo int

	StatusCode int
	Header textproto.MIMEHeader
	ContentLength int
	Body []byte
}

func Connect(uri string) (self *Client, err error) {
	var URL *url.URL
	if URL, err = url.Parse(html.UnescapeString(uri)); err != nil {
		return
	}

	if strings.IndexByte(URL.Host, ':') == -1 {
		URL.Host = URL.Host + ":554"
	}

	dailer := net.Dialer{}
	var conn net.Conn
	if conn, err = dailer.Dial("tcp", URL.Host); err != nil {
		return
	}

	u2 := *URL
	u2.User = nil

	self = &Client{
		conn: conn,
		rconn: conn,
		url: URL,
		requestUri: u2.String(),
	}
	return
}

func (self *Client) Streams() (streams []av.CodecData) {
	for _, stream := range self.streams {
		streams = append(streams, stream.CodecData)
	}
	return
}

func (self *Client) writeLine(line string) (err error) {
	if self.DebugConn {
		fmt.Print("> ", line)
	}
	_, err = fmt.Fprint(self.conn, line)
	return
}

func (self *Client) WriteRequest(req Request) (err error) {
	self.cseq++
	req.Header = append(req.Header, self.Headers...)
	req.Header = append(req.Header, fmt.Sprintf("CSeq: %d", self.cseq))
	for _, s := range self.authorization {
		req.Header = append(req.Header, "Authorization: "+s)
	}
	if err = self.writeLine(fmt.Sprintf("%s %s RTSP/1.0\r\n", req.Method, req.Uri)); err != nil {
		return
	}
	for _, v := range req.Header {
		if err = self.writeLine(fmt.Sprint(v, "\r\n")); err != nil {
			return
		}
	}
	if err = self.writeLine("\r\n"); err != nil {
		return
	}
	return
}

func (self *Client) ReadResponse() (res Response, err error) {
	var br *bufio.Reader

	defer func() {
		if br != nil {
			buf, _ := br.Peek(br.Buffered())
			self.rconn = io.MultiReader(bytes.NewReader(buf), self.rconn)
		}
		if res.StatusCode == 200 {
			if res.ContentLength > 0 {
				res.Body = make([]byte, res.ContentLength)
				if _, err = io.ReadFull(self.rconn, res.Body); err != nil {
					return
				}
			}
		} else if res.BlockLength > 0 {
			res.Block = make([]byte, res.BlockLength)
			if _, err = io.ReadFull(self.rconn, res.Block); err != nil {
				return
			}
		}
	}()

	var h [4]byte
	if _, err = io.ReadFull(self.rconn, h[:]); err != nil {
		return
	}

	if h[0] == 36 {
		// $
		res.BlockLength = int(h[2])<<8+int(h[3])
		res.BlockNo = int(h[1])
		if self.DebugConn {
			fmt.Println("block: len", res.BlockLength, "no", res.BlockNo)
		}
		return
	} else if h[0] == 82 && h[1] == 84 && h[2] == 83 && h[3] == 80 {
		// RTSP 200 OK
		self.rconn = io.MultiReader(bytes.NewReader(h[:]), self.rconn)
	} else {
		for {
			for {
				var b [1]byte
				if _, err = self.rconn.Read(b[:]); err != nil {
					return
				}
				if b[0] == 36 {
					break
				}
			}
			if self.DebugConn {
				fmt.Println("block: relocate")
			}
			if _, err = io.ReadFull(self.rconn, h[1:4]); err != nil {
				return
			}
			res.BlockLength = int(h[2])<<8+int(h[3])
			res.BlockNo = int(h[1])
			if res.BlockNo/2 < len(self.streams) {
				break
			}
		}
		if self.DebugConn {
			fmt.Println("block: len", res.BlockLength, "no", res.BlockNo)
		}
		return
	}

	br = bufio.NewReader(self.rconn)
	tp := textproto.NewReader(br)

	var line string
	if line, err = tp.ReadLine(); err != nil {
		return
	}
	if self.DebugConn {
		fmt.Println("<", line)
	}

	fline := strings.SplitN(line, " ", 3)
	if len(fline) < 2 {
		err = fmt.Errorf("malformed RTSP response line")
		return
	}

	if res.StatusCode, err = strconv.Atoi(fline[1]); err != nil {
		return
	}
	var header textproto.MIMEHeader
	if header, err = tp.ReadMIMEHeader(); err != nil {
		return
	}

	if self.DebugConn {
		fmt.Println("<", header)
	}

	if res.StatusCode != 200 && res.StatusCode != 401 {
		err = fmt.Errorf("rtsp: StatusCode=%d invalid", res.StatusCode)
		return
	}

	if res.StatusCode == 401 {
		/*
		RTSP/1.0 401 Unauthorized
		CSeq: 2
		Date: Wed, May 04 2016 10:10:51 GMT
		WWW-Authenticate: Digest realm="LIVE555 Streaming Media", nonce="c633aaf8b83127633cbe98fac1d20d87"
		*/
		authval := header.Get("WWW-Authenticate")
		hdrval := strings.SplitN(authval, " ", 2)
		var realm, nonce string

		if len(hdrval) == 2 {
			for _, field := range strings.Split(hdrval[1], ",") {
				field = strings.Trim(field, ", ")
				if keyval := strings.Split(field, "="); len(keyval) == 2 {
					key := keyval[0]
					val := strings.Trim(keyval[1], `"`)
					switch key {
					case "realm":
						realm = val
					case "nonce":
						nonce = val
					}
				}
			}

			if realm != "" && nonce != "" {
				if self.url.User == nil {
					err = fmt.Errorf("rtsp: please provide username and password")
					return
				}
				var username string
				var password string
				var ok bool
				username = self.url.User.Username()
				if password, ok = self.url.User.Password(); !ok {
					err = fmt.Errorf("rtsp: please provide password")
					return
				}
				hs1 := md5hash(username+":"+realm+":"+password)
				hs2 := md5hash("DESCRIBE:"+self.requestUri)
				response := md5hash(hs1+":"+nonce+":"+hs2)
				self.authorization = []string{
					fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
						username, realm, nonce, self.requestUri, response),
					fmt.Sprintf(`Basic %s`, base64.StdEncoding.EncodeToString([]byte(username+":"+password))),
				}
			}
		}
	}

	if sess := header.Get("Session"); sess != "" && self.session == "" {
		if fields := strings.Split(sess, ";"); len(fields) > 0 {
			self.session = fields[0]
		}
	}

	res.ContentLength, _ = strconv.Atoi(header.Get("Content-Length"))

	return
}

func (self *Client) Setup(streams []int) (err error) {
	for _, si := range streams {
		uri := ""
		control := self.streams[si].Sdp.Control
		if strings.HasPrefix(control, "rtsp://") {
			uri = control
		} else {
			uri = self.requestUri+"/"+control
		}
		req := Request{Method: "SETUP", Uri: uri}
		req.Header = append(req.Header, fmt.Sprintf("Transport: RTP/AVP/TCP;unicast;interleaved=%d-%d", si*2, si*2+1))
		if self.session != "" {
			req.Header = append(req.Header, "Session: "+self.session)
		}
		if err = self.WriteRequest(req); err != nil {
			return
		}
		if _, err = self.ReadResponse(); err != nil {
			return
		}
	}
	return
}

func md5hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func (self *Client) Describe() (streams []av.CodecData, err error) {
	var res Response

	for i := 0; i < 2; i++ {
		req := Request{
			Method: "DESCRIBE",
			Uri: self.requestUri,
		}
		if err = self.WriteRequest(req); err != nil {
			return
		}
		if res, err = self.ReadResponse(); err != nil {
			return
		}
		if res.StatusCode == 200 {
			break
		}
	}
	if res.ContentLength == 0 {
		err = fmt.Errorf("rtsp: Describe failed, StatusCode=%d", res.StatusCode)
		return
	}

	body := string(res.Body)

	if self.DebugConn {
		fmt.Println("<", body)
	}

	self.streams = []*Stream{}
	for _, info := range sdp.Decode(body) {
		stream := &Stream{Sdp: info}

		if false {
			fmt.Println("sdp:", info.TimeScale)
		}

		if info.PayloadType >= 96 && info.PayloadType <= 127 {
			switch info.Type {
			case av.H264:
				var sps, pps []byte
				for _, nalu := range info.SpropParameterSets {
					if len(nalu) > 0 {
						switch nalu[0]&0x1f {
						case 7:
							sps = nalu
						case 8:
							pps = nalu
						}
					}
				}
				if len(sps) > 0 && len(pps) > 0 {
					if stream.CodecData, err = h264parser.NewCodecDataFromSPSAndPPS(sps, pps); err != nil {
						err = fmt.Errorf("rtsp: h264 sps/pps invalid: %s", err)
						return
					}
				} else {
					err = fmt.Errorf("rtsp: h264 sdp sprop-parameter-sets invalid: missing sps or pps")
					return
				}

			case av.AAC:
				if len(info.Config) == 0 {
					err = fmt.Errorf("rtsp: aac sdp config missing")
					return
				}
				if stream.CodecData, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(info.Config); err != nil {
					err = fmt.Errorf("rtsp: aac sdp config invalid: %s", err)
					return
				}
			}
		} else {
			switch info.PayloadType {
			case 0:
				stream.CodecData = codec.NewPCMMulawCodecData()

			default:
				err = fmt.Errorf("rtsp: PayloadType=%d unsupported", info.PayloadType)
				return
			}
		}

		self.streams = append(self.streams, stream)
	}

	for _, stream := range self.streams {
		streams = append(streams, stream)
	}
	self.pktque = &pktqueue.Queue{
		Poll: self.poll,
	}
	self.pktque.Alloc(streams)

	return
}

func (self *Client) Options() (err error) {
	if err = self.WriteRequest(Request{
		Method: "OPTIONS",
		Uri: self.requestUri,
	}); err != nil {
		return
	}
	if _, err = self.ReadResponse(); err != nil {
		return
	}
	return
}

func (self *Stream) handleH264Payload(naluType byte, timestamp uint32, packet []byte) (err error) {
	/*
	Table 7-1 – NAL unit type codes
	1   ￼Coded slice of a non-IDR picture
	5    Coded slice of an IDR picture
	6    Supplemental enhancement information (SEI)
	7    Sequence parameter set
	8    Picture parameter set
	*/
	switch naluType {
	case 7,8:
		// sps/pps

	default:
		if naluType == 5 {
			self.pkt.IsKeyFrame = true
		}
		self.gotpkt = true
		self.pkt.Data = packet
		self.timestamp = timestamp
	}

	return
}

func (self *Stream) handlePacket(timestamp uint32, packet []byte) (err error) {
	switch self.Type() {
	case av.H264:
		/*
		+---------------+
		|0|1|2|3|4|5|6|7|
		+-+-+-+-+-+-+-+-+
		|F|NRI|  Type   |
		+---------------+
		*/
		naluType := packet[0]&0x1f

		/*
		NAL Unit  Packet    Packet Type Name               Section
		Type      Type
		-------------------------------------------------------------
		0        reserved                                     -
		1-23     NAL unit  Single NAL unit packet             5.6
		24       STAP-A    Single-time aggregation packet     5.7.1
		25       STAP-B    Single-time aggregation packet     5.7.1
		26       MTAP16    Multi-time aggregation packet      5.7.2
		27       MTAP24    Multi-time aggregation packet      5.7.2
		28       FU-A      Fragmentation unit                 5.8
		29       FU-B      Fragmentation unit                 5.8
		30-31    reserved                                     -
		*/

		switch {
		case naluType >= 1 && naluType <= 23:
			if err = self.handleH264Payload(naluType, timestamp, packet); err != nil {
				return
			}

		case naluType == 28: // FU-A
			/*
			0                   1                   2                   3
			0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			| FU indicator  |   FU header   |                               |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
			|                                                               |
			|                         FU payload                            |
			|                                                               |
			|                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			|                               :...OPTIONAL RTP padding        |
			+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			Figure 14.  RTP payload format for FU-A

			The FU indicator octet has the following format:
			+---------------+
			|0|1|2|3|4|5|6|7|
			+-+-+-+-+-+-+-+-+
			|F|NRI|  Type   |
			+---------------+


			The FU header has the following format:
			+---------------+
			|0|1|2|3|4|5|6|7|
			+-+-+-+-+-+-+-+-+
			|S|E|R|  Type   |
			+---------------+

			S: 1 bit
			When set to one, the Start bit indicates the start of a fragmented
			NAL unit.  When the following FU payload is not the start of a
			fragmented NAL unit payload, the Start bit is set to zero.

			E: 1 bit
			When set to one, the End bit indicates the end of a fragmented NAL
			unit, i.e., the last byte of the payload is also the last byte of
			the fragmented NAL unit.  When the following FU payload is not the
			last fragment of a fragmented NAL unit, the End bit is set to
			zero.

			R: 1 bit
			The Reserved bit MUST be equal to 0 and MUST be ignored by the
			receiver.

			Type: 5 bits
			The NAL unit payload type as defined in table 7-1 of [1].
			*/
			fuIndicator := packet[0]
			fuHeader := packet[1]
			isStart := fuHeader&0x80!=0
			isEnd := fuHeader&0x40!=0
			naluType := fuHeader&0x1f
			if isStart {
				self.fuBuffer = []byte{fuIndicator&0xe0|fuHeader&0x1f}
			}
			self.fuBuffer = append(self.fuBuffer, packet[2:]...)
			if isEnd {
				if err = self.handleH264Payload(naluType, timestamp, self.fuBuffer); err != nil {
					return
				}
			}

		case naluType == 24:
			err = fmt.Errorf("rtsp: unsupported H264 STAP-A")
			return

		default:
			err = fmt.Errorf("rtsp: unsupported H264 naluType=%d", naluType)
			return
		}

	case av.AAC:
		self.gotpkt = true
		self.pkt.Data = packet[4:]
		self.timestamp = timestamp

	default:
		self.gotpkt = true
		self.pkt.Data = packet
		self.timestamp = timestamp
	}
	return
}

func (self *Client) parseBlock(blockNo int, packet []byte) (streamIndex int, err error) {
	if blockNo % 2 != 0 {
		// rtcp block
		return
	}

	streamIndex = blockNo/2
	if streamIndex >= len(self.streams) {
		err = fmt.Errorf("rtsp: parseBlock: streamIndex=%d invalid", streamIndex)
		return
	}
	stream := self.streams[streamIndex]

	/*
	0                   1                   2                   3
	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|V=2|P|X|  CC   |M|     PT      |       sequence number         |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                           timestamp                           |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|           synchronization source (SSRC) identifier            |
	+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
	|            contributing source (CSRC) identifiers             |
	|                             ....                              |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	*/

	if len(packet) < 8 {
		err = fmt.Errorf("rtp packet too short")
		return
	}
	payloadOffset := 12+int(packet[0]&0xf)*4
	if payloadOffset+2 > len(packet) {
		err = fmt.Errorf("rtp packet too short")
		return
	}

	timestamp := binary.BigEndian.Uint32(packet[4:8])
	payload := packet[payloadOffset:]

	/*
	PT 	Encoding Name 	Audio/Video (A/V) 	Clock Rate (Hz) 	Channels 	Reference 
	0	PCMU	A	8000	1	[RFC3551]
	1	Reserved				
	2	Reserved				
	3	GSM	A	8000	1	[RFC3551]
	4	G723	A	8000	1	[Vineet_Kumar][RFC3551]
	5	DVI4	A	8000	1	[RFC3551]
	6	DVI4	A	16000	1	[RFC3551]
	7	LPC	A	8000	1	[RFC3551]
	8	PCMA	A	8000	1	[RFC3551]
	9	G722	A	8000	1	[RFC3551]
	10	L16	A	44100	2	[RFC3551]
	11	L16	A	44100	1	[RFC3551]
	12	QCELP	A	8000	1	[RFC3551]
	13	CN	A	8000	1	[RFC3389]
	14	MPA	A	90000		[RFC3551][RFC2250]
	15	G728	A	8000	1	[RFC3551]
	16	DVI4	A	11025	1	[Joseph_Di_Pol]
	17	DVI4	A	22050	1	[Joseph_Di_Pol]
	18	G729	A	8000	1	[RFC3551]
	19	Reserved	A			
	20	Unassigned	A			
	21	Unassigned	A			
	22	Unassigned	A			
	23	Unassigned	A			
	24	Unassigned	V			
	25	CelB	V	90000		[RFC2029]
	26	JPEG	V	90000		[RFC2435]
	27	Unassigned	V			
	28	nv	V	90000		[RFC3551]
	29	Unassigned	V			
	30	Unassigned	V			
	31	H261	V	90000		[RFC4587]
	32	MPV	V	90000		[RFC2250]
	33	MP2T	AV	90000		[RFC2250]
	34	H263	V	90000		[Chunrong_Zhu]
	35-71	Unassigned	?			
	72-76	Reserved for RTCP conflict avoidance				[RFC3551]
	77-95	Unassigned	?			
	96-127	dynamic	?			[RFC3551]
	*/
	//payloadType := packet[1]&0x7f

	if self.DebugConn {
		//fmt.Println("packet:", stream.Type(), "offset", payloadOffset, "pt", payloadType)
		if len(packet)>24 {
			fmt.Println(hex.Dump(packet[:24]))
		}
	}

	if err = stream.handlePacket(timestamp, payload); err != nil {
		return
	}

	return
}

func (self *Client) Play() (err error) {
	req := Request{
		Method: "PLAY",
		Uri: self.requestUri,
	}
	req.Header = append(req.Header, "Session: "+self.session)
	if err = self.WriteRequest(req); err != nil {
		return
	}
	return
}

func (self *Client) poll() (err error) {
	for {
		var res Response
		if res, err = self.ReadResponse(); err != nil {
			return
		}
		if res.BlockLength > 0 {
			var i int
			if i, err = self.parseBlock(res.BlockNo, res.Block); err != nil {
				return
			}
			stream := self.streams[i]
			if stream.gotpkt {
				time := float64(stream.timestamp)/float64(stream.Sdp.TimeScale)
				if false {
					fmt.Printf("rtsp: #%d %d/%d %d\n", i, stream.timestamp, stream.Sdp.TimeScale, len(stream.pkt.Data))
				}
				self.pktque.WriteTimePacket(i, time, stream.pkt)
				stream.pkt = av.Packet{}
				stream.gotpkt = false
				return
			}
		}
	}
	return
}

func (self *Client) ReadPacket() (i int, pkt av.Packet, err error) {
	return self.pktque.ReadPacket()
}

func Open(uri string) (cli *Client, err error) {
	_cli, err := Connect(uri)
	if err != nil {
		return
	}

	streams, err := _cli.Describe()
	if err != nil {
		return
	}

	setup := []int{}
	for i := range streams {
		setup = append(setup, i)
	}

	err = _cli.Setup(setup)
	if err != nil {
		return
	}

	err = _cli.Play()
	if err != nil {
		return
	}

	cli = _cli
	return
}

