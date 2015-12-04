
package main

import (
	"bytes"
	"os"
	"io"
	ts "../"
	"fmt"
	"encoding/hex"
)

type Stream struct {
	PID uint
	Header *ts.PESHeader
	Title string
	Data bytes.Buffer
}

func main() {
	var file *os.File
	var err error
	if file, err = os.Open("/tmp/out.ts"); err != nil {
		return
	}

	data := [188]byte{}

	var n int
	var header ts.TSHeader
	var pat ts.PAT
	var pmt ts.PMT
	var payload []byte
	var info ts.ElementaryStreamInfo
	streams := map[uint]*Stream{}

	findOrCreateStream := func(pid uint) (stream *Stream) {
		stream, _ = streams[pid]
		if stream == nil {
			stream = &Stream{
				PID: pid,
			}
			if info.StreamType == ts.ElementaryStreamTypeH264 {
				stream.Title = "h264"
			} else if info.StreamType == ts.ElementaryStreamTypeAdtsAAC {
				stream.Title = "aac"
			}
			streams[pid] = stream
		}
		return
	}

	onStreamPayload := func() (err error) {
		stream := findOrCreateStream(header.PID)
		r := bytes.NewReader(payload)
		lr := &io.LimitedReader{R: r, N: int64(len(payload))}
		if header.PayloadUnitStart {
			stream.Data = bytes.Buffer{}
			if stream.Header, err = ts.ReadPESHeader(lr); err != nil {
				return
			}
		}
		if _, err = io.CopyN(&stream.Data, lr, lr.N); err != nil {
			return
		}
		if stream.Data.Len() == int(stream.Header.DataLength) {
			fmt.Println(stream.Title, stream.Data.Len(), "total")
			fmt.Println(hex.Dump(stream.Data.Bytes()))
		}
		return
	}

	for {
		if header, n, err = ts.ReadTSPacket(file, data[:]); err != nil {
			return
		}
		fmt.Println(header, n)

		payload = data[:n]
		pr := bytes.NewReader(payload)

		if header.PID == 0 {
			if pat, err = ts.ReadPAT(pr); err != nil {
				return
			}
		}

		for _, entry := range(pat.Entries) {
			if entry.ProgramMapPID == header.PID {
				//fmt.Println("matchs", entry)
				if pmt, err = ts.ReadPMT(pr); err != nil {
					return
				}
				//fmt.Println("pmt", pmt)
				if true {
					fmt.Println(hex.Dump(payload))
				}
			}
		}

		for _, info = range(pmt.ElementaryStreamInfos) {
			if info.ElementaryPID == header.PID {
				onStreamPayload()
			}
		}

	}
}

