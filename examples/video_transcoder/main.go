package main

import (
	"fmt"
	"sync"
	"io"
	"net/http"
	"os"

	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/av/transcode"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/nareix/joy4/format/flv"
	"github.com/nareix/joy4/cgo/ffmpeg"
)

func init() {
	format.RegisterAll()
}

type writeFlusher struct {
	httpflusher http.Flusher
	io.Writer
}

func (w writeFlusher) Flush() error {
	w.httpflusher.Flush()
	return nil
}

func printHelp() {
	fmt.Printf ("Usage: ./video_normalizer <video_file> \n")
}

func main() {
	fmt.Println("starting server")
	server := &rtmp.Server{}
	videoFile := ""

	if len(os.Args) <= 1 {
		printHelp()
		return
	}

	for i, arg := range os.Args {
		if i == 0 {
			// skip program name
			continue
		}
		if arg == "-help" || arg == "-h" {
			printHelp()
			return
		} else {
			videoFile = os.Args[i]
		}
	}

	if videoFile == "" {
		printHelp()
		return
	}

	fmt.Println("Video file:", videoFile, ". Play with: ffplay http://localhost:8089/file")

	l := &sync.RWMutex{}
	type Channel struct {
		que *pubsub.Queue
	}
	channels := map[string]*Channel{}

	server.HandlePlay = func(conn *rtmp.Conn) {
		fmt.Println("HandlePlay()")
		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		if ch != nil {
			cursor := ch.que.Latest()
			avutil.CopyFile(conn, cursor)
		}
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		fmt.Println("HandlePublish()")
		streams, _ := conn.Streams()

		l.Lock()
		ch := channels[conn.URL.Path]
		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			ch.que.WriteHeader(streams)
			channels[conn.URL.Path] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			return
		}

		trans := &transcode.Demuxer{
			Options: transcode.Options{
				FindAudioDecoderEncoder: FindAudioCodec,
				FindVideoDecoderEncoder: FindVideoCodec,
			},
			Demuxer: conn,
		}

		avutil.CopyFile(ch.que, trans)

		fmt.Println("Leaving HandlePublish()")

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()
		ch.que.Close()
	}

	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request:", r.URL.Path)
		w.Header().Set("Content-Type", "video/x-flv")
		w.Header().Set("Transfer-Encoding", "chunked")		
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)
		flusher.Flush()

		file, _ := avutil.Open(videoFile)

		trans := &transcode.Demuxer{
			Options: transcode.Options{
				FindAudioDecoderEncoder: FindAudioCodec,
				FindVideoDecoderEncoder: FindVideoCodec,
			},
			Demuxer: file,
		}

		muxer := flv.NewMuxerWriteFlusher(writeFlusher{httpflusher: flusher, Writer: w})
		avutil.CopyFile(muxer, trans)
		file.Close()
	})

	go http.ListenAndServe(":8089", nil)
	server.ListenAndServe()
	fmt.Println("Done")
}

// FindAudioCodec is a callback used by joy4's transcoder to find an audio codec compatible with the input stream
func FindAudioCodec(stream av.AudioCodecData, i int) (need bool, dec av.AudioDecoder, enc av.AudioEncoder, err error) {
	need = true
	dec, err = ffmpeg.NewAudioDecoder(stream)
	if err != nil {
		return
	}
	if dec == nil {
		err = fmt.Errorf("Audio decoder not found")
		return
	}

	enc, err = ffmpeg.NewAudioEncoderByCodecType(av.AAC)
	if err != nil {
		return
	}
	if enc == nil {
		err = fmt.Errorf("Audio encoder not found")
		return
	}
	enc.SetSampleRate(44100)
	enc.SetChannelLayout(av.CH_STEREO)
	enc.SetBitrate(192000)
	enc.SetOption("profile", "HE-AACv2")
	return
}

// FindVideoCodec is a callback used by joy4's transcoder to find a video codec compatible with the input stream
func FindVideoCodec(stream av.VideoCodecData, i int) (need bool, dec *ffmpeg.VideoDecoder, enc *ffmpeg.VideoEncoder, err error) {
	need = true
	dec, err = ffmpeg.NewVideoDecoder(stream)
	if err != nil {
		return
	}
	if dec == nil {
		err = fmt.Errorf("Video decoder not found")
		return
	}

	enc, err = ffmpeg.NewVideoEncoderByCodecType(av.H264)
	if err != nil {
		return
	}
	if enc == nil {
		err = fmt.Errorf("Video encoder not found")
		return
	}

	// Encoder config
	FpsNum := 25000
	FpsDen := 1000
	// Configurable (can be set from input stream, or set by user and the input video will be converted before encoding)
	enc.SetFramerate(FpsNum, FpsDen)
	enc.SetResolution(640, 480)
	enc.SetPixelFormat(av.I420)
	// Must be configured by user
	enc.SetBitrate(1000000) // 1 Mbps
	enc.SetGopSize(FpsNum / FpsDen) // 1s gop
	return
}
