package main

import (
	"fmt"
	"math"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/cgo/ffmpeg"
)

type Normalizer struct {
	Audio  audioProfile
	Video  videoProfile
}

type audioProfile struct {
	inputConfig     av.AudioConfig
	needsTranscode  bool
	transcodeConfig av.AudioConfig
}

type videoProfile struct {
	inputConfig     av.VideoConfig
	needsTranscode  bool
	transcodeConfig av.VideoConfig
}

// TODO ?
// type StreamConfig interface {
// }

type resolutionConfig struct {
	Width, Height  int
	stride, hAlign int
	configs        map[string]*av.VideoConfig
}

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
		fmt.Printf("audio CodecType unsupported: transcode %v => %v\n", aInput.CodecType, aTranscode.CodecType)
	}

	if aInput.Format != audioConfig.Format {
		aTranscode.Format = audioConfig.Format
		n.Audio.needsTranscode = true
		fmt.Printf("audio Format unsupported: transcode %v => %v\n", aInput.Format, aTranscode.Format)
	}

	if aInput.SampleRate != audioConfig.SampleRate {
		aTranscode.SampleRate = audioConfig.SampleRate
		n.Audio.needsTranscode = true
		fmt.Printf("audio SampleRate unsupported: transcode %v => %v\n", aInput.SampleRate, aTranscode.SampleRate)
	}

	if aInput.Layout != audioConfig.Layout {
		aTranscode.Layout = audioConfig.Layout
		n.Audio.needsTranscode = true
		fmt.Printf("audio Layout unsupported: transcode %v => %v\n", aInput.Layout, aTranscode.Layout)
	}

	// TODO check:
	//	bitrate
	return
}

func (n *Normalizer) NormalizeVideoProfile(stream av.CodecData) (err error) {
	// Init transcode config with input properties
	n.Video.transcodeConfig = n.Video.inputConfig

	// Check the streams against accepted configs
	// to determine what needs to be transcoded
	_, videoConfigs := getAVConfigs()

	vInput := &n.Video.inputConfig
	vTranscode := &n.Video.transcodeConfig

	vInput.FpsNum, vInput.FpsDen = fixFps(vInput.FpsNum, vInput.FpsDen)

	var resolutionConfig *resolutionConfig
	var videoConfig *av.VideoConfig
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
		fmt.Printf("\033[43mvideo resolution %vx%v not supported => Force 480p\033[0m\n", vInput.Width, vInput.Height)
		// TODO define scaling here (including stretching to a supported format)

		resolutionConfig = videoConfigs["480"]
		n.Video.needsTranscode = true
	}

	if resolutionConfig != nil {
		fmt.Printf("\033[42mvideo resolution %vx%v supported\033[0m\n", resolutionConfig.Width, resolutionConfig.Height)

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
		fmt.Println(err)
		n.Video.needsTranscode = true
		return err
	} else if mustConvertFramerate {
		fmt.Printf("\033[43mvideo fps %v/%v not supported for this resolution => convert to %v/%v (fps diff: %v)\033[0m\n", vInput.FpsNum, vInput.FpsDen, bestFpsMatchNum, bestFpsMatchDen, bestFpsMatchDiff)
		fmt.Printf("\033[43mfps convertion is not ready yet !\033[0m\n") // FIXME
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
		fmt.Printf("video CodecType unsupported: transcode %v => %v\n", vInput.CodecType, vTranscode.CodecType)
	}

	// TODO check:
	//	bitrate
	//	YStride, CStride int
	//	SubsampleRatio image.YCbCrSubsampleRatio
	//	ProfileIdc, LevelIdc uint
	//	ScanningMode
	//	Bitdepth, via pixelformat ?

	return
}

// FindAudioCodec is a callback used by joy4's transcoder to find an audio codec compatible with the input stream
func (n *Normalizer) FindAudioCodec(stream av.AudioCodecData, i int) (need bool, dec av.AudioDecoder, enc av.AudioEncoder, err error) {
	err = n.NormalizeAudioProfile(stream)

	if err != nil {
		fmt.Println(err)
		return true, nil, nil, err
	}

	if !n.Audio.needsTranscode {
		fmt.Printf( "Audio transcode not needed. config: %+v\n", n.Audio.inputConfig)
		return false, nil, nil, err
	}

	fmt.Printf( "Audio transcode needed. config: %+v\n", n.Audio.inputConfig)

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

	enc, err = ffmpeg.NewAudioEncoderByCodecType(n.Audio.transcodeConfig.CodecType)
	if err != nil {
		fmt.Println(err)
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
	// TODO replace profile with 'n', too many structs and types !!
	// TODO config.String() avec la res et le fps, minimum

	err = n.NormalizeVideoProfile(stream)

	if err != nil {
		fmt.Println(err)
		return true, nil, nil, err
	}

	if !n.Video.needsTranscode {
		fmt.Printf( "Video transcode not needed. config: %+v\n", n.Video.inputConfig)
		return false, nil, nil, err
	}

	fmt.Printf( "Video transcode needed. config: %+v\n", n.Video.inputConfig)

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

	enc, err = ffmpeg.NewVideoEncoderByCodecType(n.Video.transcodeConfig.CodecType)
	if err != nil {
		fmt.Println(err)
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
func getAVConfigs() (audioConfig av.AudioConfig, videoConfig map[string]*resolutionConfig) {
	audioConfig.CodecType = av.AAC
	audioConfig.Format = av.FLTP
	audioConfig.SampleRate = 44100
	audioConfig.Layout = av.CH_STEREO

	videoConfig = make(map[string]*resolutionConfig)
	videoConfig["1080"] = &resolutionConfig{Width: 1920, Height: 1080, configs: make(map[string]*av.VideoConfig)}
	videoConfig["720"] = &resolutionConfig{Width: 1280, Height: 720, configs: make(map[string]*av.VideoConfig)}
	videoConfig["480"] = &resolutionConfig{Width: 640, Height: 480, configs: make(map[string]*av.VideoConfig)}
	videoConfig["360"] = &resolutionConfig{Width: 640, Height: 360, configs: make(map[string]*av.VideoConfig)}
	videoConfig["160"] = &resolutionConfig{Width: 240, Height: 160, configs: make(map[string]*av.VideoConfig)}

	// TODO fill resolutionConfig stride, hAlign
	// TODO check 480p, 360p and 160p standard
	// TODO fill one and deep-copy to other instead of init everything

	videoConfig["1080"].configs["60"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 60000, FpsDen: 1000,
	}
	videoConfig["1080"].configs["50"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 50000, FpsDen: 1000,
	}
	videoConfig["1080"].configs["30"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfig["1080"].configs["25"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1920, Height: 1080,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfig["720"].configs["60"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 60000, FpsDen: 1000,
	}
	videoConfig["720"].configs["50"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 50000, FpsDen: 1000,
	}
	videoConfig["720"].configs["30"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfig["720"].configs["25"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     1280, Height: 720,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfig["480"].configs["30"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     640, Height: 480,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfig["480"].configs["25"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     640, Height: 480,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfig["360"].configs["30"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     640, Height: 360,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfig["360"].configs["25"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     640, Height: 360,
		FpsNum: 25000, FpsDen: 1000,
	}

	videoConfig["160"].configs["30"] = &av.VideoConfig{
		CodecType: av.H264,
		Width:     240, Height: 160,
		FpsNum: 30000, FpsDen: 1000,
	}
	videoConfig["160"].configs["25"] = &av.VideoConfig{
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

// TODO rename
func fixFps(num, den int) (n, d int) {
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
