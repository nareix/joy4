package main

import (
	"fmt"
	"math"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/cgo/ffmpeg"
)


type RateControlMode uint8
const (
	CBR = RateControlMode(iota + 1)
	VBR
)

type RateControl struct {
	bitrateKbps int
	rateControlMode RateControlMode
}

type ScanningMode uint8
const (
	Progressive = ScanningMode(iota + 1)
	InterlacedTopFieldFirst
	InterlacedBottomFieldFirst
)

type AudioStream struct {
	inputConfig AudioConfig
	needsTranscode bool
	transcodeConfig AudioConfig
}

type VideoStream struct {
	inputConfig VideoConfig
	needsTranscode bool
	transcodeConfig VideoConfig
}

type AudioConfig struct {
	codecType av.CodecType
	rc RateControl
	format av.SampleFormat
	isLittleEndian bool
	sampleRate int
	layout av.ChannelLayout
}

type VideoConfig struct {
	codecType av.CodecType
	rc RateControl
	width, height int
	stride, hAlign int
	fpsNum, fpsDen int
	pixFmt av.PixelFormat
	scanningMode ScanningMode
}

type ResolutionConfig struct {
	width, height, stride, hAlign int
	configs map[string]*VideoConfig
}

	// TODO configs must be externalized
func getAVConfigs() (audioConfig AudioConfig, videoConfig map[string]*ResolutionConfig){
	fmt.Println("getAVConfigs()")

	audioConfig.codecType		= av.AAC
	audioConfig.rc				= RateControl{ bitrateKbps: 320, rateControlMode: CBR }
	audioConfig.format			= av.S16 // TODO FLTP ?
	audioConfig.isLittleEndian	= true // irrelevant if fltp ?
	audioConfig.sampleRate		= 44100
	audioConfig.layout			= av.CH_STEREO


	videoConfig = make(map[string]*ResolutionConfig)
	videoConfig["1080"] = &ResolutionConfig{width: 1920, height: 1080, stride: 1920, hAlign: 1088, configs: make(map[string]*VideoConfig)}
	videoConfig[ "720"] = &ResolutionConfig{width: 1280, height:  720, stride: 1280, hAlign:  720, configs: make(map[string]*VideoConfig)}
	videoConfig[ "480"] = &ResolutionConfig{width:  640, height:  480, stride:  640, hAlign:  480, configs: make(map[string]*VideoConfig)}
	videoConfig[ "360"] = &ResolutionConfig{width:  640, height:  360, stride:  640, hAlign:  360, configs: make(map[string]*VideoConfig)}
	videoConfig[ "160"] = &ResolutionConfig{width:  240, height:  160, stride:  240, hAlign:  160, configs: make(map[string]*VideoConfig)}

	// TODO pixFMT I420 or something else ?
	// TODO check 480p, 360p and 160p standard
	// TODO adjust bitrates
	// TODO fill one and deep-copy to other instead of init everything

	videoConfig["1080"].configs["60"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 12000, rateControlMode: CBR},
		width: 1920, height: 1080, stride: 1920, hAlign: 1088,
		fpsNum: 60000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig["1080"].configs["50"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 11000, rateControlMode: CBR},
		width: 1920, height: 1080, stride: 1920, hAlign: 1088,
		fpsNum: 50000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig["1080"].configs["30"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 8000, rateControlMode: CBR},
		width: 1920, height: 1080, stride: 1920, hAlign: 1088,
		fpsNum: 30000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig["1080"].configs["25"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 7000, rateControlMode: CBR},
		width: 1920, height: 1080, stride: 1920, hAlign: 1088,
		fpsNum: 25000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}

	videoConfig[ "720"].configs["60"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 8000, rateControlMode: CBR},
		width: 1280, height: 720, stride: 1280, hAlign: 720,
		fpsNum: 60000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig[ "720"].configs["50"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 7000, rateControlMode: CBR},
		width: 1280, height: 720, stride: 1280, hAlign: 720,
		fpsNum: 50000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig[ "720"].configs["30"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 5000, rateControlMode: CBR},
		width: 1280, height: 720, stride: 1280, hAlign: 720,
		fpsNum: 30000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig[ "720"].configs["25"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 4000, rateControlMode: CBR},
		width: 1280, height: 720, stride: 1280, hAlign: 720,
		fpsNum: 25000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}

	videoConfig[ "480"].configs["30"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 3000, rateControlMode: CBR},
		width: 640, height: 480, stride: 640, hAlign: 480,
		fpsNum: 30000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig[ "480"].configs["25"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 2000, rateControlMode: CBR},
		width: 640, height: 480, stride: 640, hAlign: 480,
		fpsNum: 25000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}

	videoConfig[ "360"].configs["30"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 1500, rateControlMode: CBR},
		width: 640, height: 360, stride: 640, hAlign: 360,
		fpsNum: 30000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig[ "360"].configs["25"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 1000, rateControlMode: CBR},
		width: 640, height: 360, stride: 640, hAlign: 360,
		fpsNum: 25000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}

	videoConfig[ "160"].configs["30"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 800, rateControlMode: CBR},
		width: 240, height: 160, stride: 240, hAlign: 160,
		fpsNum: 30000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}
	videoConfig[ "160"].configs["25"] = &VideoConfig{
		codecType: av.H264, rc: RateControl{ bitrateKbps: 600, rateControlMode: CBR},
		width: 240, height: 160, stride: 240, hAlign: 160,
		fpsNum: 25000, fpsDen: 1000, pixFmt: av.I420, scanningMode: Progressive,
	}

	return
}

func gcd(a, b int) int {
    for b > 0 {
        a, b = b, a%b
    }
    return a
}

func fixFps(num, den int) (n, d int) {
	g := gcd(num, den)
	n = num/g
	d = den/g

	if d == 1 {
		d *= 1000
		n *= 1000
	}

	// fmt.Printf("in: %v/%v => gdc: %v  out: %v/%v\n", num, den, g, n, d)
	return
}


func probeAudioStream(stream av.CodecData) (a AudioStream, err error) {
	// Fill inputConfig from input stream
	// TODO fill rc, isLittleEndian
	a.inputConfig.codecType		= stream.Type()
	a.inputConfig.format		= stream.(av.AudioCodecData).SampleFormat()
	a.inputConfig.sampleRate	= stream.(av.AudioCodecData).SampleRate()
	a.inputConfig.layout		= stream.(av.AudioCodecData).ChannelLayout()

	// Init transcode config with input properties
	a.transcodeConfig = a.inputConfig


	// Check the streams against accepted configs
	// to determine what needs to be transcoded
	audioConfig, _ := getAVConfigs()

	aInput := &a.inputConfig
	aTranscode := &a.transcodeConfig

	if aInput.codecType != audioConfig.codecType {
		aTranscode.codecType = audioConfig.codecType
		a.needsTranscode = true
		fmt.Printf("audio codecType unsupported: transcode %v => %v\n", aInput.codecType, aTranscode.codecType)
	}

	if aInput.rc.bitrateKbps > audioConfig.rc.bitrateKbps {
		aTranscode.rc.bitrateKbps = audioConfig.rc.bitrateKbps
		a.needsTranscode = true
		fmt.Printf("audio rc.bitrateKbps unsupported: transcode %v => %v\n", aInput.rc.bitrateKbps, aTranscode.rc.bitrateKbps)
	}

	if aInput.rc.rateControlMode != audioConfig.rc.rateControlMode {
		aTranscode.rc.rateControlMode = audioConfig.rc.rateControlMode
		a.needsTranscode = true
		fmt.Printf("audio rc.rateControlMode unsupported: transcode %v => %v\n", aInput.rc.rateControlMode, aTranscode.rc.rateControlMode)
	}

	if aInput.format != audioConfig.format {
		aTranscode.format = audioConfig.format
		a.needsTranscode = true
		fmt.Printf("audio format unsupported: transcode %v => %v\n", aInput.format, aTranscode.format)
	}

	if aInput.isLittleEndian != audioConfig.isLittleEndian {
		aTranscode.isLittleEndian = audioConfig.isLittleEndian
		a.needsTranscode = true
		fmt.Printf("audio isLittleEndian unsupported: transcode %v => %v\n", aInput.isLittleEndian, aTranscode.isLittleEndian)
	}

	if aInput.sampleRate != audioConfig.sampleRate {
		aTranscode.sampleRate = audioConfig.sampleRate
		a.needsTranscode = true
		fmt.Printf("audio sampleRate unsupported: transcode %v => %v\n", aInput.sampleRate, aTranscode.sampleRate)
	}

	if aInput.layout != audioConfig.layout {
		aTranscode.layout = audioConfig.layout
		a.needsTranscode = true
		fmt.Printf("audio layout unsupported: transcode %v => %v\n", aInput.layout, aTranscode.layout)
	}
	return
}

func probeVideoStream(stream av.CodecData) (v VideoStream, err error) {
	// Fill inputConfig from input stream
	// TODO fill rc, stride, hAlign, pixFmt, scanningMode
	v.inputConfig.codecType						= stream.Type()
	v.inputConfig.width							= stream.(av.VideoCodecData).Width()
	v.inputConfig.height						= stream.(av.VideoCodecData).Height()
	v.inputConfig.fpsNum, v.inputConfig.fpsDen	= stream.(av.VideoCodecData).Framerate()

	// Init transcode config with input properties
	v.transcodeConfig = v.inputConfig


	// Check the streams against accepted configs
	// to determine what needs to be transcoded
	_, videoConfigs := getAVConfigs()

	vInput := &v.inputConfig
	vTranscode := &v.transcodeConfig
	var videoConfigForRes *VideoConfig
	videoConfigFound := false

	vInput.fpsNum, vInput.fpsDen = fixFps(vInput.fpsNum, vInput.fpsDen)
	fmt.Printf("\033[35mvideo input: %vx%v, %v/%v\033[0m\n", vInput.width, vInput.height, vInput.fpsNum, vInput.fpsDen)


	for _, resolutionConfigs := range(videoConfigs) {
		if vInput.width  == resolutionConfigs.width && vInput.height == resolutionConfigs.height {

			var bestFpsMatchNum int
			var bestFpsMatchDen int
			var bestFpsMatchDiff float64

			for _, fpsConfig := range(resolutionConfigs.configs) {
				fmt.Printf("\033[36mchecking against: %vx%v %v/%v\033[0m\n", resolutionConfigs.width, resolutionConfigs.height, fpsConfig.fpsNum, fpsConfig.fpsDen)

				if vInput.fpsNum == fpsConfig.fpsNum && vInput.fpsDen == fpsConfig.fpsDen {
					fmt.Println("found")
					videoConfigForRes = fpsConfig
					videoConfigFound = true
					break
				} else {
					fpsInputVal	:= float64(vInput.fpsNum)    / float64(vInput.fpsDen)
					fpsVal		:= float64(fpsConfig.fpsNum) / float64(fpsConfig.fpsDen)
					fpsDiff		:= math.Abs(fpsInputVal - fpsVal)

					if bestFpsMatchDiff == 0 || fpsDiff < bestFpsMatchDiff {
						bestFpsMatchNum = fpsConfig.fpsNum
						bestFpsMatchDen = fpsConfig.fpsDen
						bestFpsMatchDiff = fpsDiff
						fmt.Printf("\033[37mBest candidate: %v/%v (fps diff: %v)\033[0m\n", bestFpsMatchNum, bestFpsMatchDen, bestFpsMatchDiff)
					}
				}
			}

			if !videoConfigFound {
				fmt.Println("\033[34mthe framerate must be converted from", vInput.fpsNum, vInput.fpsDen, "to", bestFpsMatchNum, bestFpsMatchDen, "\033[0m")
			} else {
				fmt.Println("resolution and framerate are accepted")
				break
			}
		}
	}

	// TODO fill transcodeConfig FPS for framerate conversion
	// TODO video stretching to a supported format

	if !videoConfigFound {
		err = fmt.Errorf("Video standard not supported: %dx%d, %d/%d", vInput.width, vInput.height, vInput.fpsNum, vInput.fpsDen)
		fmt.Println(err)
		v.needsTranscode = true
		return v, err
	}

	if vInput.codecType != videoConfigForRes.codecType {
		vTranscode.codecType = videoConfigForRes.codecType
		v.needsTranscode = true
		fmt.Printf("video codecType unsupported: transcode %v => %v\n", vInput.codecType, vTranscode.codecType)
	}

	if vInput.rc.bitrateKbps > videoConfigForRes.rc.bitrateKbps {
		vTranscode.rc.bitrateKbps = videoConfigForRes.rc.bitrateKbps
		v.needsTranscode = true
		fmt.Printf("video rc.bitrateKbps unsupported: transcode %v => %v\n", vInput.rc.bitrateKbps, vTranscode.rc.bitrateKbps)
	}

	if vInput.rc.rateControlMode != videoConfigForRes.rc.rateControlMode {
		vTranscode.rc.rateControlMode = videoConfigForRes.rc.rateControlMode
		v.needsTranscode = true
		fmt.Printf("video rc.rateControlMode unsupported: transcode %v => %v\n", vInput.rc.rateControlMode, vTranscode.rc.rateControlMode)
	}

	if vInput.stride != videoConfigForRes.stride {
		vTranscode.stride = videoConfigForRes.stride
		v.needsTranscode = true
		fmt.Printf("video stride unsupported: transcode %v => %v\n", vInput.stride, vTranscode.stride)
	}

	if vInput.hAlign != videoConfigForRes.hAlign {
		vTranscode.hAlign = videoConfigForRes.hAlign
		v.needsTranscode = true
		fmt.Printf("video hAlign unsupported: transcode %v => %v\n", vInput.hAlign, vTranscode.hAlign)
	}

	if vInput.pixFmt != videoConfigForRes.pixFmt {
		vTranscode.pixFmt = videoConfigForRes.pixFmt
		v.needsTranscode = true
		fmt.Printf("video pixFmt unsupported: transcode %v => %v\n", vInput.pixFmt, vTranscode.pixFmt)
	}

	if vInput.scanningMode != videoConfigForRes.scanningMode {
		vTranscode.scanningMode = videoConfigForRes.scanningMode
		v.needsTranscode = true
		fmt.Printf("video scanningMode unsupported: transcode %v => %v\n", vInput.scanningMode, vTranscode.scanningMode)
	}

	return
}



func findAudioCodec(stream av.AudioCodecData, i int) (need bool, dec av.AudioDecoder, enc av.AudioEncoder, err error) {
	var a AudioStream
	a, err = probeAudioStream(stream)

	if err != nil {
		fmt.Println(err)
		return true, nil, nil, err
	}

	if !a.needsTranscode {
		fmt.Printf("\033[31mAudio transcode not needed\033[0m\n")
		return false, nil, nil, err
	}

	fmt.Printf("\033[31mAudio transcode needed !!!\033[0m\n")
	
	need = true
	dec, err = ffmpeg.NewAudioDecoder(stream)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dec == nil {
		err = fmt.Errorf("Audio decoder not found")
		return
	}

	enc, err = ffmpeg.NewAudioEncoderByCodecType(a.transcodeConfig.codecType)
	if err != nil {
		fmt.Println(err)
		return
	}
	if enc == nil {
		err = fmt.Errorf("Audio encoder not found")
		return
	}
	enc.SetSampleRate(a.transcodeConfig.sampleRate)
	enc.SetChannelLayout(a.transcodeConfig.layout)
	enc.SetBitrate(a.transcodeConfig.rc.bitrateKbps * 1000)
	enc.SetOption("profile", "HE-AACv2") // TODO should be in audioConfig
	return
}


func findVideoCodec(stream av.VideoCodecData, i int) (need bool, dec av.VideoDecoder, enc av.VideoEncoder, err error) {
	var v VideoStream
	v, err = probeVideoStream(stream)

	if err != nil {
		fmt.Println(err)
		return true, nil, nil, err
	}

	if !v.needsTranscode {
		fmt.Printf("\033[32mVideo transcode not needed\033[0m\n")
		return false, nil, nil, err
	}


	fmt.Printf("\033[32mVideo transcode needed !!!\033[0m\n")

	need = true
	dec, err = ffmpeg.NewVideoDecoder(stream)
	if err != nil {
		fmt.Println(err)
		return
	}
	if dec == nil {
		err = fmt.Errorf("Video decoder not found")
		return
	}

	enc, err = ffmpeg.NewVideoEncoderByCodecType(v.transcodeConfig.codecType)
	if err != nil {
		fmt.Println(err)
		return
	}
	if enc == nil {
		err = fmt.Errorf("Video encoder not found")
		return
	}

	fpsNum := v.transcodeConfig.fpsNum
	fpsDen := v.transcodeConfig.fpsDen
	fmt.Println("input fps:", fpsNum, fpsDen)

	// Encoder config
	// Must be set from input stream
	enc.SetFramerate(fpsNum, fpsDen) // TODO set from transcodeConfig
	// Configurable (can be set from input stream, or set by user and the input video will be converted before encoding)
	enc.SetResolution(v.transcodeConfig.width, v.transcodeConfig.height)
	enc.SetPixelFormat(v.transcodeConfig.pixFmt)
	// Must be configured by user
	enc.SetBitrate(v.transcodeConfig.rc.bitrateKbps * 1000)
	enc.SetGopSize(fpsNum/fpsDen) // 1s gop TODO should be in videoConfig
	return
}
