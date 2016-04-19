package main

import (
	ts "../"
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
)

type GobAllSamples struct {
	TimeScale int
	SPS       []byte
	PPS       []byte
	Samples   []GobSample
}

type GobSample struct {
	Duration int
	Data     []byte
	Sync     bool
}

type Stream struct {
	PID           uint
	PESHeader     *ts.PESHeader
	FirstTSHeader ts.TSHeader
	Title         string
	Data          bytes.Buffer
	Type          uint
	PCR           uint64
}

type Sample struct {
	Type                  uint
	PCR                   uint64
	PTS                   uint64
	DTS                   uint64
	Data                  []byte
	RandomAccessIndicator bool
}

var (
	debugData   = true
	debugStream = true
)

func readSamples(filename string, ch chan Sample) {
	defer func() {
		close(ch)
	}()

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
				PID:  pid,
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
		if debugStream {
			fmt.Printf("stream: %s end\n", stream.Title)
		}
		if debugData {
			fmt.Println(stream.Type, stream.Title, stream.Data.Len(), "total")
			fmt.Println(hex.Dump(stream.Data.Bytes()))
		}
		ch <- Sample{
			Type: stream.Type,
			Data: stream.Data.Bytes(),
			PTS:  stream.PESHeader.PTS,
			DTS:  stream.PESHeader.DTS,
			PCR:  stream.FirstTSHeader.PCR,
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
				fmt.Printf("stream: %s start\n", stream.Title)
			}
		}

		if _, err = io.CopyN(&stream.Data, lr, lr.N); err != nil {
			return
		}
		if debugStream {
			fmt.Printf("stream: %s %d/%d\n", stream.Title, stream.Data.Len(), stream.PESHeader.DataLength)
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
			fmt.Println("header:", header, n)
		}

		payload = data[:n]
		pr := bytes.NewReader(payload)

		if header.PID == 0 {
			if pat, err = ts.ReadPAT(pr); err != nil {
				return
			}
		}

		for _, entry := range pat.Entries {
			if entry.ProgramMapPID == header.PID {
				//fmt.Println("matchs", entry)
				if pmt, err = ts.ReadPMT(pr); err != nil {
					return
				}
				//fmt.Println("pmt", pmt)
			}
		}

		for _, info = range pmt.ElementaryStreamInfos {
			if info.ElementaryPID == header.PID {
				onStreamPayload()
			}
		}

	}
}

func writeM3U8Header(w io.Writer) {
	fmt.Fprintln(w, `#EXTM3U
#EXT-X-ALLOW-CACHE:YES
#EXT-X-PLAYLIST-TYPE:VOD
#EXT-X-TARGETDURATION:9
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:0`)
}

func writeM3U8Item(w io.Writer, filename string, size int64, duration float64) {
	fmt.Fprintf(w, `#EXT-X-BYTE-SIZE:%d
#EXTINF:%f,
%s
`, size, duration, filename)
}

func writeM3U8Footer(w io.Writer) {
	fmt.Fprintln(w, `#EXT-X-ENDLIST`)
}

func testInputGob(pathGob string, pathOut string, testSeg bool, writeM3u8 bool) {
	var m3u8file *os.File
	lastFilename := pathOut

	gobfile, _ := os.Open(pathGob)
	outfile, _ := os.Create(pathOut)
	dec := gob.NewDecoder(gobfile)
	var allSamples GobAllSamples
	dec.Decode(&allSamples)

	if writeM3u8 {
		m3u8file, _ = os.Create("index.m3u8")
		writeM3U8Header(m3u8file)
	}

	muxer := &ts.Muxer{
		W: outfile,
	}
	trackH264 := muxer.AddH264Track()
	trackH264.SPS = allSamples.SPS
	trackH264.PPS = allSamples.PPS
	trackH264.TimeScale = int64(allSamples.TimeScale)
	muxer.WriteHeader()

	lastPTS := int64(0)
	syncCount := 0
	segCount := 0

	for i, sample := range allSamples.Samples {
		if debugStream {
			fmt.Println("stream: write sample #", i)
		}
		if sample.Sync {
			syncCount++
			if testSeg {
				if syncCount%3 == 0 {
					filename := fmt.Sprintf("%s.seg%d.ts", pathOut, segCount)

					if debugStream {
						fmt.Println("stream:", "seg", segCount, "sync", syncCount, trackH264.PTS)
					}

					if m3u8file != nil {
						info, _ := outfile.Stat()
						size := info.Size()
						dur := float64(trackH264.PTS-lastPTS) / float64(allSamples.TimeScale)
						writeM3U8Item(m3u8file, lastFilename, size, dur)
					}

					lastFilename = filename
					outfile.Close()
					segCount++
					outfile, _ = os.Create(filename)
					muxer.W = outfile
					muxer.WriteHeader()
					lastPTS = trackH264.PTS
				}
			}
		}

		trackH264.WriteH264NALU(sample.Sync, sample.Duration, sample.Data)
	}

	if m3u8file != nil {
		writeM3U8Footer(m3u8file)
		m3u8file.Close()
	}

	outfile.Close()
	if debugStream {
		fmt.Println("stream: written to", pathOut)
	}
}

func main() {
	input := flag.String("i", "", "input file")
	output := flag.String("o", "", "output file")
	inputGob := flag.String("g", "", "input gob file")
	testSegment := flag.Bool("seg", false, "test segment")
	writeM3u8 := flag.Bool("m3u8", false, "write m3u8 file")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")

	flag.BoolVar(&debugData, "vd", false, "debug data")
	flag.BoolVar(&debugStream, "vs", false, "debug stream")
	flag.BoolVar(&ts.DebugReader, "vr", false, "debug reader")
	flag.BoolVar(&ts.DebugWriter, "vw", false, "debug writer")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *inputGob != "" && *output != "" {
		testInputGob(*inputGob, *output, *testSegment, *writeM3u8)
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
				{StreamType: ts.ElementaryStreamTypeAdtsAAC, ElementaryPID: 0x101},
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
		var streamId uint
		var pid uint

		switch sample.Type {
		case ts.ElementaryStreamTypeH264:
			streamId = ts.StreamIdH264
			pid = 0x100
		case ts.ElementaryStreamTypeAdtsAAC:
			streamId = ts.StreamIdAAC
			pid = 0x101
		}

		pes := ts.PESHeader{
			StreamId: streamId,
			PTS:      sample.PTS,
			DTS:      sample.DTS,
		}
		w.PID = pid
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
			W:   file,
			PID: 0x100,
		}
		w.EnableVecWriter()
	}

	for {
		var ok bool
		if sample, ok = <-ch; !ok {
			break
		}

		if debugStream {
			fmt.Println("sample: ", sample.Type, len(sample.Data),
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
