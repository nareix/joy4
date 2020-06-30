package main

import (
	"github.com/sprucehealth/joy4/av"
	"github.com/sprucehealth/joy4/av/avutil"
	"github.com/sprucehealth/joy4/av/transcode"
	"github.com/sprucehealth/joy4/cgo/ffmpeg"
	"github.com/sprucehealth/joy4/format"
	"github.com/sprucehealth/joy4/format/rtmp"
)

// need ffmpeg with libspeex and libfdkaac installed
//
// open http://www.wowza.com/resources/4.4.1/examples/WebcamRecording/FlashRTMPPlayer11/player.html
// click connect and recored
// input camera H264/SPEEX will converted H264/AAC and saved in out.ts

func init() {
	format.RegisterAll()
}

func main() {
	server := &rtmp.Server{}

	server.HandlePublish = func(conn *rtmp.Conn) {
		file, _ := avutil.Create("out.ts")

		findcodec := func(stream av.AudioCodecData, i int) (need bool, dec av.AudioDecoder, enc av.AudioEncoder, err error) {
			need = true
			dec, _ = ffmpeg.NewAudioDecoder(stream)
			enc, _ = ffmpeg.NewAudioEncoderByName("libfdk_aac")
			enc.SetSampleRate(48000)
			enc.SetChannelLayout(av.CH_STEREO)
			return
		}

		trans := &transcode.Demuxer{
			Options: transcode.Options{
				FindAudioDecoderEncoder: findcodec,
			},
			Demuxer: conn,
		}

		avutil.CopyFile(file, trans)
	}

	server.ListenAndServe()
}
