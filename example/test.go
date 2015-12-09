
package main

import (
	"bytes"
	"os"
	"io"
	ts "../"
	"fmt"
	"encoding/hex"
	"encoding/gob"
	"runtime/pprof"
	"flag"
)

type GobAllSamples struct {
	TimeScale int
	SPS []byte
	PPS []byte
	Samples []GobSample
}

type GobSample struct {
	Duration int
	Data []byte
	Sync bool
}

type Stream struct {
	PID uint
	PESHeader *ts.PESHeader
	FirstTSHeader ts.TSHeader
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
	RandomAccessIndicator bool
}

func readSamples(filename string, ch chan Sample) {
	defer func() {
		close(ch)
	}()

	debugData := true
	debugStream := true

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

	onStreamPayloadUnitEnd := func(stream *Stream) {
		if debugData {
			fmt.Println(stream.Type, stream.Title, stream.Data.Len(), "total")
			fmt.Println(hex.Dump(stream.Data.Bytes()))
		}
		ch <- Sample{
			Type: stream.Type,
			Data: stream.Data.Bytes(),
			PTS: stream.PESHeader.PTS,
			DTS: stream.PESHeader.DTS,
			PCR: stream.FirstTSHeader.PCR,
			RandomAccessIndicator: stream.FirstTSHeader.RandomAccessIndicator,
		}
	}

	onStreamPayload := func() (err error) {
		stream := findOrCreateStream(header.PID)
		r := bytes.NewReader(payload)
		lr := &io.LimitedReader{R: r, N: int64(len(payload))}

		if header.PayloadUnitStart && stream.PESHeader != nil && stream.PESHeader.DataLength == 0 {
			onStreamPayloadUnitEnd(stream)
		}

		if header.PayloadUnitStart {
			stream.Data = bytes.Buffer{}
			if stream.PESHeader, err = ts.ReadPESHeader(lr); err != nil {
				return
			}
			stream.FirstTSHeader = header
			if debugStream {
				fmt.Printf("stream: start\n")
			}
		}

		if _, err = io.CopyN(&stream.Data, lr, lr.N); err != nil {
			return
		}

		if debugStream {
			fmt.Printf("stream: %d/%d\n", stream.Data.Len(), stream.PESHeader.DataLength)
		}

		if stream.Data.Len() == int(stream.PESHeader.DataLength) {
			onStreamPayloadUnitEnd(stream)
		}

		return
	}

	for {
		if header, n, err = ts.ReadTSPacket(file, data[:]); err != nil {
			return
		}
		if debugData {
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
			}
		}

		for _, info = range(pmt.ElementaryStreamInfos) {
			if info.ElementaryPID == header.PID {
				onStreamPayload()
			}
		}

	}
}

func testInputGob(pathGob string, pathOut string) {
	gobfile, _ := os.Open(pathGob)
	outfile, _ := os.Create(pathOut)
	dec := gob.NewDecoder(gobfile)
	var allSamples GobAllSamples
	dec.Decode(&allSamples)

	w := ts.SimpleH264Writer{
		W: outfile,
		SPS: allSamples.SPS,
		PPS: allSamples.PPS,
		TimeScale: allSamples.TimeScale,
	}

	for _, sample := range allSamples.Samples {
		w.WriteNALU(sample.Sync, sample.Duration, sample.Data)
	}

	outfile.Close()
	fmt.Println("written to", pathOut)
}

func main() {
	input := flag.String("i", "", "input file")
	output := flag.String("o", "", "output file")
	inputGob := flag.String("g", "", "input gob file")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}


	ts.DebugReader = true
	ts.DebugWriter = true

	if *inputGob != "" && *output != "" {
		testInputGob(*inputGob, *output)
		return
	}

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
		pat := ts.PAT{
			Entries: []ts.PATEntry{
				{ProgramNumber: 1, ProgramMapPID: 0x1000},
			},
		}
		if err = ts.WritePATPacket(file, pat); err != nil {
			return
		}
		return
	}

	writePMT := func() (err error) {
		pmt := ts.PMT{
			PCRPID: 0x100,
			ElementaryStreamInfos: []ts.ElementaryStreamInfo{
				{StreamType: ts.ElementaryStreamTypeH264, ElementaryPID: 0x100},
			},
		}
		if err = ts.WritePMTPacket(file, pmt, 0x1000); err != nil {
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
		w.RandomAccessIndicator = sample.RandomAccessIndicator
		if err = ts.WritePESPacket(w, pes, sample.Data); err != nil {
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
					"DTS", sample.DTS, "sync", sample.RandomAccessIndicator,
				)
				//fmt.Print(hex.Dump(sample.Data))
			}

			if file != nil {
				writeSample()
			}
		}
	}

}

