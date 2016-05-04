package rtsp

import (
	"fmt"
	"net"
	"io"
	"io/ioutil"
	"strings"
	"strconv"
	"bufio"
	"net/textproto"
	"net/url"
	"encoding/hex"
	"crypto/md5"
	"github.com/nareix/av"
)

type Client struct {
	DebugConn bool
	url *url.URL
	conn net.Conn
	requestUri string
	cseq uint
	streams []*Stream
	session string
	authorization string
	body io.Reader
}

func Connect(uri string) (self *Client, err error) {
	var URL *url.URL
	if URL, err = url.Parse(uri); err != nil {
		return
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
		url: URL,
		requestUri: u2.String(),
	}
	return
}

func (self *Client) writeLine(line string) (err error) {
	if self.DebugConn {
		fmt.Print(line)
	}
	_, err = fmt.Fprint(self.conn, line)
	return
}

func (self *Client) WriteRequest(method string, uri string, headers []string) (err error) {
	self.cseq++
	headers = append(headers, fmt.Sprintf("CSeq: %d", self.cseq))
	if err = self.writeLine(fmt.Sprintf("%s %s RTSP/1.0\r\n", method, uri)); err != nil {
		return
	}
	for _, header := range headers {
		if err = self.writeLine(header+"\r\n"); err != nil {
			return
		}
	}
	if err = self.writeLine("\r\n"); err != nil {
		return
	}
	return
}

func (self *Client) ReadResponse() (statusCode int, body io.Reader, err error) {
	br := bufio.NewReader(self.conn)
	tp := textproto.NewReader(br)

	var line string
	if line, err = tp.ReadLine(); err != nil {
		return
	}
	if self.DebugConn {
		fmt.Println(line)
	}

	fline := strings.SplitN(line, " ", 3)
	if len(fline) < 2 {
		err = fmt.Errorf("malformed RTSP response")
		return
	}

	if statusCode, err = strconv.Atoi(fline[1]); err != nil {
		return
	}
	if statusCode != 200 && statusCode != 401 {
		err = fmt.Errorf("statusCode(%d) invalid", statusCode)
		return
	}

	var header textproto.MIMEHeader
	if header, err = tp.ReadMIMEHeader(); err != nil {
		return
	}

	if statusCode == 401 {
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
					err = fmt.Errorf("please provide username and password")
					return
				}
				var username string
				var password string
				var ok bool
				username = self.url.User.Username()
				if password, ok = self.url.User.Password(); !ok {
					err = fmt.Errorf("please provide password")
					return
				}
				hs1 := md5hash(username+":"+realm+":"+password)
				hs2 := md5hash("DESCRIBE:"+self.requestUri)
				response := md5hash(hs1+":"+nonce+":"+hs2)
				self.authorization = fmt.Sprintf(
						`Authorization: Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
						username, realm, nonce, self.requestUri, response)
			}
		}
	}

	if sess := header.Get("Session"); sess != "" && self.session == "" {
		if fields := strings.Split(sess, ";"); len(fields) > 0 {
			self.session = fields[0]
		}
	}

	clen, _ := strconv.Atoi(header.Get("Content-Length"))

	if statusCode == 200 {
		if clen > 0 {
			body = io.LimitReader(br, int64(clen))
		} else {
			body = io.MultiReader(io.LimitReader(br, int64(br.Buffered())), self.conn)
		}
	}

	return
}

func (self *Client) Setup(streams []int) (err error) {
	for _, si := range streams {
		reqhdr := []string{fmt.Sprintf("Transport: RTP/AVP/TCP;unicast;interleaved=%d-%d", si*2, si*2+1)}
		if self.session != "" {
			reqhdr = append(reqhdr, "Session: "+self.session)
		}
		if err = self.WriteRequest("SETUP", self.requestUri+"/"+self.streams[si].control, reqhdr); err != nil {
			return
		}
		if _, _, err = self.ReadResponse(); err != nil {
			return
		}
	}
	return
}

func md5hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func (self *Client) Describe() (streams []*Stream, err error) {
	var body io.Reader
	var statusCode int

	for i := 0; i < 2; i++ {
		reqhdr := []string{}
		if self.authorization != "" {
			reqhdr = append(reqhdr, self.authorization)
		}
		if err = self.WriteRequest("DESCRIBE", self.requestUri, reqhdr); err != nil {
			return
		}
		if statusCode, body, err = self.ReadResponse(); err != nil {
			return
		}
		if statusCode == 200 {
			break
		}
	}
	if body == nil {
		err = fmt.Errorf("Describe failed")
		return
	}

	br := bufio.NewReader(body)
	tp := textproto.NewReader(br)
	var stream *Stream

	for {
		line, err := tp.ReadLine()
		if err != nil {
			break
		}

		if self.DebugConn {
			fmt.Println(line)
		}

		typeval := strings.SplitN(line, "=", 2)
		if len(typeval) == 2 {
			fields := strings.Split(typeval[1], " ")
			switch typeval[0] {
			case "m":
				if len(fields) > 0 {
					switch fields[0] {
					case "audio", "video":
						stream = &Stream{typestr: fields[0]}
						self.streams = append(self.streams, stream)
					}
				}

			case "a":
				if stream != nil {
					for _, field := range fields {
						keyval := strings.Split(field, ":")
						if len(keyval) >= 2 {
							key := keyval[0]
							val := keyval[1]
							if key == "control" {
								stream.control = val
							}
						}
					}
				}
			}
		}
	}

	streams = self.streams
	return
}

func (self *Client) Options() (err error) {
	if err = self.WriteRequest("OPTIONS", self.requestUri, []string{}); err != nil {
		return
	}
	if _, _, err = self.ReadResponse(); err != nil {
		return
	}
	return
}

func (self *Client) readBlock() (err error) {
	var h [4]byte
	for {
		if _, err = io.ReadFull(self.body, h[:]); err != nil {
			return
		}
		if h[0] != 36 {
			err = fmt.Errorf("block not start with $")
			fmt.Println(h)
			return
		}
		length := int(h[2])<<8+int(h[3])

		if self.DebugConn {
			fmt.Println("packet", length, h[1])
		}

		if _, err = io.CopyN(ioutil.Discard, self.body, int64(length)); err != nil {
			return
		}
	}
}

func (self *Client) ReadHeader() (streams []av.Stream, err error) {
	if err = self.WriteRequest("PLAY", self.requestUri, []string{"Session: "+self.session}); err != nil {
		return
	}
	if _, self.body, err = self.ReadResponse(); err != nil {
		return
	}

	for {
		if err = self.readBlock(); err != nil {
			return
		}
	}
	return
}

