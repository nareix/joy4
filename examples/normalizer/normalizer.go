package main

import (
	"fmt"
	"image"
	"math"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/cgo/ffmpeg"
)

// Normalizer contains the audio an video profiles (input characteristics, and desired output characteristics)
type Normalizer struct {
	Audio  audioProfile
	Video  videoProfile
}

type audioProfile struct {
	inputConfig     audioConfig
	needsTranscode  bool
	transcodeConfig audioConfig
}

type videoProfile struct {
	inputConfig     videoConfig
	needsTranscode  bool
	transcodeConfig videoConfig
}

type audioConfig struct {
	CodecType  av.CodecType
	Bitrate    av.BitrateMeasure
	Format     av.SampleFormat
	SampleRate int
	Layout     av.ChannelLayout
}

type videoConfig struct {
	CodecType        av.CodecType
	Bitrate          av.BitrateMeasure
	Width, Height    int
	FpsNum, FpsDen   int
	YStride, CStride int
	SubsampleRatio   image.YCbCrSubsampleRatio
	H264CodecData    av.CodecData // h264parser.CodecData
	ScanMode         av.ScanningMode
}

type resolutionConfig struct {
	Width, Height  int
	stride, hAlign int
	configs        map[string]*videoConfig
}

// NormalizeAudioProfile checks compatibility between n.Audio.inputConfig and n.Audio.transcodeConfig.
// After returning, needsTranscode says if the input stream should be transcoded to the transcodeConfig.
func (n *Normalizer) NormalizeAudioProfile(stream av.CodecData) (err error) {
	// Init transcode config with input properties
	n.Audio.transcodeConfig = n.Audio.inputConfig

	// Check the streams against accepted configs
	// to determine what needs to be transcoded
	audioConfig, _ := getAVConfigs()

	aInput := &n.Audio.inputConfig
	aTranscode := &n.Audio.transcodeConfig

	if aInput.CodecType != audioConfig.CodecType {
		aTranscode.CodecType = audioConfig.CodecType
		n.Audio.needsTranscode = true
		fmt.Println(
			"Audio CodecType unsupported",
			"aInput.CodecType", aInput.CodecType,
			"aTranscode.CodecType", aTranscode.CodecType,
		)
	}

	if aInput.Format != audioConfig.Format {
		aTranscode.Format = audioConfig.Format
		n.Audio.needsTranscode = true
		fmt.Println(
			"Audio Format unsupported",
			"aInput.Format", aInput.Format,
			"aTranscode.Format", aTranscode.Format,
		)
	}

	if aInput.SampleRate != audioConfig.SampleRate {
		aTranscode.SampleRate = audioConfig.SampleRate
		n.Audio.needsTranscode = true
		fmt.Println(
			"Audio Sample Rate unsupported",
			"aInput.SampleRate", aInput.SampleRate,
			"aTranscode.SampleRate", aTranscode.SampleRate,
		)
	}

	if aInput.Layout != audioConfig.Layout {
		aTranscode.Layout = audioConfig.Layout
		n.Audio.needsTranscode = true
		fmt.Println(
			"Audio Layout unsupported",
			"aInput.Layout", aInput.Layout,
			"aTranscode.Layout", aTranscode.Layout,
		)
	}

	// TODO check:
	//	bitrate
	return
}

// NormalizeVideoProfile checks compatibility between n.Video.inputConfig and n.Video.transcodeConfig.
// After returning, needsTranscode says if the input stream should be transcoded to the transcodeConfig.
func (n *Normalizer) NormalizeVideoProfile(stream av.CodecData) (err error) {
	// Init transcode config with input properties
	n.Video.transcodeConfig = n.Video.inputConfig

	// Check the streams against accepted configs
	// to determine what needs to be transcoded
	_, videoConfigs := getAVConfigs()

	vInput := &n.Video.inputConfig
	vTranscode := &n.Video.transcodeConfig

	vInput.FpsNum, vInput.FpsDen = reduceFpsFraction(vInput.FpsNum, vInput.FpsDen)

	var resolutionConfig *resolutionConfig
	var videoConfig *videoConfig
	var mustConvertFramerate bool
	var bestFpsMatchNum int
	var bestFpsMatchDen int
	var bestFpsMatchDiff float64

	for _, resolutionConfigs := range videoConfigs {
		if vInput.Width == resolutionConfigs.Width && vInput.Height == resolutionConfigs.Height {
			resolutionConfig = resolutionConfigs
		}
	}

	if resolutionConfig == nil {
		fmt.Println(
			"Video Resolution not supported, force 480p",
			"vInput.Width", vInput.Width,
			"vInput.Height", vInput.Height,
		)
		// TODO define scaling here (including stretching to a supported format)

		resolutionConfig = videoConfigs["480"]
		n.Video.needsTranscode = true
	}

	if resolutionConfig != nil {
		fmt.Println(
			"Video Resolution supported",
			"resolutionConfig.Width", resolutionConfig.Width,
			"resolutionConfig.Height", resolutionConfig.Height,
		)

		for _, fpsConfig := range resolutionConfig.configs {
			if vInput.FpsNum == fpsConfig.FpsNum && vInput.FpsDen == fpsConfig.FpsDen {
				// There is an existing fpsConfig, just use it, no need to convert the framerate
				videoConfig = fpsConfig
				break
			} else {
				// The fpsConfig doesn't match, compute if it could be a good match for framerate conversion
				fpsInputVal := float64(vInput.FpsNum) / float64(vInput.FpsDen)
				fpsVal := float64(fpsConfig.FpsNum) / float64(fpsConfig.FpsDen)
				fpsDiff := math.Abs(fpsInputVal - fpsVal)

				if bestFpsMatchDiff == 0 || fpsDiff < bestFpsMatchDiff {
					bestFpsMatchNum = fpsConfig.FpsNum
					bestFpsMatchDen = fpsConfig.FpsDen
					bestFpsMatchDiff = fpsDiff
					videoConfig = fpsConfig
					mustConvertFramerate = true
				}
			}
		}
	}

	if videoConfig == nil {
		err = fmt.Errorf("video standard not supported: %dx%d, %d/%d", vInput.Width, vInput.Height, vInput.FpsNum, vInput.FpsDen)
		fmt.Println(
			"Video standard not supported",
			"vInput.Width", vInput.Width,
			"vInput.Height", vInput.Height,
			"vInput.FpsNum", vInput.FpsNum,
			"vInput.FpsDen", vInput.FpsDen,
		)
		n.Video.needsTranscode = true
		return err
	} else if mustConvertFramerate {
		fmt.Println(
			"Video fps not supported for this resolution, must be converted to best fps match",
			"vInput.FpsNum", vInput.FpsNum,
			"vInput.FpsDen", vInput.FpsDen,
			"bestFpsMatchNum", bestFpsMatchNum,
			"bestFpsMatchDen", bestFpsMatchDen,
			"bestFpsMatchDiff", bestFpsMatchDiff,
		)
		// FIXME
		fmt.Println("Framerate convertion is not ready yet !")
		n.Video.transcodeConfig = *videoConfig

		// vTranscode.FpsNum = bestFpsMatchNum
		// vTranscode.FpsDen = bestFpsMatchDen
		// n.Video.needsTranscode = true
	} else {
		fmt.Println("Input framerate is supported")
	}

	if vInput.CodecType != videoConfig.CodecType {
		vTranscode.CodecType = videoConfig.CodecType
		n.Video.needsTranscode = true
		fmt.Println(
			"Video CodecType unsupported",
			"vInput.CodecType", vInput.CodecType,
			"vTranscode.CodecType", vTranscode.CodecType,
		)
	}

	// TODO check:
	//	bitrate
	//	YStride, CStride int
	//	SubsampleRatio image.YCbCrSubsampleRatio
	//	ProfileIdc, LevelIdc uint
	//	ScanningMode
	//	Bitdepth, via pixelformat ?
	// H264CodecData if not nil

	return
}

// FindAudioCodec is a callback used by joy4's transcoder to find an audio codec compatible with the input stream
func (n *Normalizer) FindAudioCodec(stream av.AudioCodecData, i int) (need bool, dec av.AudioDecoder, enc av.AudioEncoder, err error) {
	err = n.NormalizeAudioProfile(stream)

	if err != nil {
		return true, nil, nil, err
	}

	if !n.Audio.needsTranscode {
		fmt.Println(
			"Audio transcode not needed",
			"config", n.Audio.inputConfig,
		)
		return false, nil, nil, err
	}

	fmt.Println(
		"Audio transcode needed",
		"config", n.Audio.transcodeConfig,
	)

	need = true
	dec, err = ffmpeg.NewAudioDecoder(stream)
	if err != nil {
		return
	}
	if dec == nil {
		err = fmt.Errorf("Audio decoder not found")
		return
	}

	enc, err = ffmpeg.NewAudioEncoderByCodecType(n.Audio.transcodeConfig.CodecType)
	if err != nil {
		return
	}
	if enc == nil {
		err = fmt.Errorf("Audio encoder not found")
		return
	}
	enc.SetSampleRate(n.Audio.transcodeConfig.SampleRate)
	enc.SetChannelLayout(n.Audio.transcodeConfig.Layout)
	enc.SetBitrate(192)
	enc.SetOption("n.Audio", "HE-AACv2") // TODO should be in audioConfig
	return
}

// FindVideoCodec is a callback used by joy4's transcoder to find a video codec compatible with the input stream
func (n *Normalizer) FindVideoCodec(stream av.VideoCodecData, i int) (need bool, dec *ffmpeg.VideoDecoder, enc *ffmpeg.VideoEncoder, err error) {
	err = n.NormalizeVideoProfile(stream)

	if err != nil {
		return true, nil, nil, err
	}

	if !n.Video.needsTranscode {
		fmt.Println(
			"Video transcode not needed",
			"config", n.Video.inputConfig,
		)
		return false, nil, nil, err
	}

	fmt.Println(
		"Video transcode needed",
		"config", n.Video.transcodeConfig,
	)

	need = true
	dec, err = ffmpeg.NewVideoDecoder(stream)
	if err != nil {
		return
	}
	if dec == nil {
		err = fmt.Errorf("Video decoder not found")
		return
	}

	enc, err = ffmpeg.NewVideoEncoderByCodecType(n.Video.transcodeConfig.CodecType)
	if err != nil {
		return
	}
	if enc == nil {
		err = fmt.Errorf("Video encoder not found")
		return
	}

	FpsNum := n.Video.transcodeConfig.FpsNum
	FpsDen := n.Video.transcodeConfig.FpsDen

	// Encoder config
	// Must be set from input stream
	enc.SetFramerate(FpsNum, FpsDen)
	// Configurable (can be set from input stream, or set by user and the input video will be converted before encoding)
	enc.SetResolution(n.Video.transcodeConfig.Width, n.Video.transcodeConfig.Height)
	enc.SetPixelFormat(av.I420)
	// Must be configured by user
	enc.SetBitrate(5000000)
	enc.SetGopSize(FpsNum / FpsDen) // 1s gop
	return
}

// TODO configs must be externalized
func getAVConfigs() (audioConfig audioConfig, videoConfigs map[string]*resolutionConfig) {
	audioConfig.CodecType = av.AAC
	audioConfig.Format = av.FLTP
	audioConfig.SampleRate = 44100
	audioConfig.Layout = av.CH_STEREO

	videoConfigs = make(map[string]*resolutionConfig)
	videoConfigs["1080"] = &resolutionConfig{Width: 1920, Height: 1080, configs: make(map[string]*videoConfig)}
	videoConfigs["720"] = &resolutionConfig{Width: 1280, Height: 720, configs: make(map[string]*videoConfig)}
	videoConfigs["480"] = &resolutionConfig{Width: 640, Height: 480, configs: make(map[string]*videoConfig)}
	videoConfigs["360"] = &resolutionConfig{Width: 640, Height: 360, configs: make(map[string]*videoConfig)}
	videoConfigs["160"] = &resolutionConfig{Width: 240, Height: 160, configs: make(map[string]*videoConfig)}

	// TODO fill resolutionConfig stride, hAlign
	// TODO check 480p, 360p and 160p standard
	// TODO fill one and deep-copy to other instead of init everything

	videoConfigs["1080"].configs["60"] = &videoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 60000, FpsDen: 1000,
	}
	videoConfigs["1080"].configs["50"] = &videoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 50000, FpsDen: 1000,
	}
	videoConfigs["1080"].configs["30"] = &videoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfigs["1080"].configs["25"] = &videoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfigs["720"].configs["60"] = &videoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 60000, FpsDen: 1000,
	}
	videoConfigs["720"].configs["50"] = &videoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 50000, FpsDen: 1000,
	}
	videoConfigs["720"].configs["30"] = &videoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfigs["720"].configs["25"] = &videoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfigs["480"].configs["30"] = &videoConfig{
		CodecType: av.H264,
		Width:     640, Height: 480,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfigs["480"].configs["25"] = &videoConfig{
		CodecType: av.H264,
		Width:     640, Height: 480,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfigs["360"].configs["30"] = &videoConfig{
		CodecType: av.H264,
		Width:     640, Height: 360,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfigs["360"].configs["25"] = &videoConfig{
		CodecType: av.H264,
		Width:     640, Height: 360,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfigs["160"].configs["30"] = &videoConfig{
		CodecType: av.H264,
		Width:     240, Height: 160,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfigs["160"].configs["25"] = &videoConfig{
		CodecType: av.H264,
		Width:     240, Height: 160,
		FpsNum: 25000, FpsDen: 1000,
	}

	return
}

func gcd(a, b int) int {
	for b > 0 {
		a, b = b, a%b
	}
	return a
}

func reduceFpsFraction(num, den int) (n, d int) {
	g := gcd(num, den)
	if g <= 0 {
		return num, den
	}
	n = num / g
	d = den / g

	if d == 1 {
		d *= 1000
		n *= 1000
	}
	return
}
