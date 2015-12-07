
package main

import (
	"bytes"
	"os"
	"io"
	ts "../"
	"fmt"
	"encoding/hex"
	"flag"
)

type Stream struct {
	PID uint
	Header *ts.PESHeader
	Title string
	Data bytes.Buffer
	Type uint
	PCR uint64
}

type Sample struct {
	Type uint
	PCR uint64
	PTS uint64
	DTS uint64
	Data []byte
}

func readSamples(filename string, ch chan Sample) {
	defer func() {
		close(ch)
	}()

	debug := false

	var file *os.File
	var err error
	if file, err = os.Open(filename); err != nil {
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
				Type: info.StreamType,
			}
			if stream.Type == ts.ElementaryStreamTypeH264 {
				stream.Title = "h264"
			} else if stream.Type == ts.ElementaryStreamTypeAdtsAAC {
				stream.Title = "aac"
			}
			streams[pid] = stream
		}
		return
	}

	debugStream := true

	onStreamPayload := func() (err error) {
		stream := findOrCreateStream(header.PID)
		r := bytes.NewReader(payload)
		lr := &io.LimitedReader{R: r, N: int64(len(payload))}

		if header.PayloadUnitStart {
			stream.Data = bytes.Buffer{}
			if stream.Header, err = ts.ReadPESHeader(lr); err != nil {
				return
			}
			stream.PCR = header.PCR
		}

		if _, err = io.CopyN(&stream.Data, lr, lr.N); err != nil {
			return
		}

		if debugStream {
			fmt.Printf("stream: %d/%d\n", stream.Data.Len(), stream.Header.DataLength)
		}

		if stream.Data.Len() == int(stream.Header.DataLength) {
			if debug {
				fmt.Println(stream.Type, stream.Title, stream.Data.Len(), "total")
				fmt.Println(hex.Dump(stream.Data.Bytes()))
			}
			ch <- Sample{
				Type: stream.Type,
				Data: stream.Data.Bytes(),
				PTS: stream.Header.PTS,
				DTS: stream.Header.DTS,
				PCR: stream.PCR,
			}
		}
		return
	}

	for {
		if header, n, err = ts.ReadTSPacket(file, data[:]); err != nil {
			return
		}
		if debug {
			fmt.Println(header, n)
		}

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
				if debug {
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

func main() {
	input := flag.String("i", "", "input file")
	output := flag.String("o", "", "output file")
	flag.Parse()

	var file *os.File
	var err error

	ch := make(chan Sample, 0)
	go readSamples(*input, ch)

	if *output != "" {
		if file, err = os.Create(*output); err != nil {
			return
		}
	}

	writePAT := func() (err error) {
		w := &ts.TSWriter{
			W: file,
			PID: 0,
			DisableHeaderPadding: true,
		}
		pat := ts.PAT{
			Entries: []ts.PATEntry{
				{ProgramNumber: 1, ProgramMapPID: 0x1000},
			},
		}
		bw := &bytes.Buffer{}
		if err = ts.WritePAT(bw, pat); err != nil {
			return
		}
		if err = w.Write(bw.Bytes(), false); err != nil {
			return
		}
		return
	}

	writePMT := func() (err error) {
		w := &ts.TSWriter{
			W: file,
			PID: 0x1000,
			DisableHeaderPadding: true,
		}
		pmt := ts.PMT{
			PCRPID: 0x100,
			ElementaryStreamInfos: []ts.ElementaryStreamInfo{
				{StreamType: ts.ElementaryStreamTypeH264, ElementaryPID: 0x100},
			},
		}
		bw := &bytes.Buffer{}
		if err = ts.WritePMT(bw, pmt); err != nil {
			return
		}
		if err = w.Write(bw.Bytes(), false); err != nil {
			return
		}
		return
	}

	var w *ts.TSWriter
	var sample Sample
	writeSample := func() (err error) {
		pes := ts.PESHeader{
			StreamId: ts.StreamIdH264,
			PTS: sample.PTS,
			DTS: sample.DTS,
		}
		w.PCR = sample.PCR
		bw := &bytes.Buffer{}
		if err = ts.WritePES(bw, pes, bytes.NewReader(sample.Data)); err != nil {
			return
		}
		if err = w.Write(bw.Bytes(), false); err != nil {
			return
		}
		return
	}

	if file != nil {
		writePAT()
		writePMT()
		w = &ts.TSWriter{
			W: file,
			PID: 0x100,
		}
	}

	for {
		var ok bool
		if sample, ok = <-ch; !ok {
			break
		}
		if sample.Type == ts.ElementaryStreamTypeH264 {
			if true {
				fmt.Println("sample: ", len(sample.Data),
					"PCR", sample.PCR, "PTS", sample.PTS,
					"DTS", sample.DTS,
				)
				//fmt.Print(hex.Dump(sample.Data))
			}

			if file != nil {
				writeSample()
			}
		}
	}

}

