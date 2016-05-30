package ffmpeg

import (
	/*
	#include "ffmpeg.h"
	int wrap_avcodec_decode_audio4(AVCodecContext *ctx, AVFrame *frame, void *data, int size, int *got) {
		struct AVPacket pkt = {.data = data, .size = size};
		return avcodec_decode_audio4(ctx, frame, got, &pkt);
	}
	void set_sample_fmt(AVCodecContext *ctx, int sample_fmt) {
		ctx->sample_fmt = sample_fmt;
	}
	*/
	"C"
	"unsafe"
	"runtime"
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/codec/aacparser"
)

type ffctx struct {
	ff C.FFCtx
}

type Resampler struct {
	InSampleFormat, OutSampleFormat av.SampleFormat
	InChannelLayout, OutChannelLayout av.ChannelLayout
	InSampleRate, OutSampleRate int
	avr *C.AVAudioResampleContext
	inframe, outframe *C.AVFrame
}

func freeResampler(self *Resampler) {
	C.avresample_free(&self.avr)
}

func (self *Resampler) Setup() (err error) {
	avr := C.avresample_alloc_context()
	C.av_opt_set_int(avr, C.CString("in_channel_layout"), C.int64_t(channelLayoutAV2FF(self.InChannelLayout)), 0)
	C.av_opt_set_int(avr, C.CString("out_channel_layout"), C.int64_t(channelLayoutAV2FF(self.OutChannelLayout)), 0)
	C.av_opt_set_int(avr, C.CString("in_sample_rate"), C.int64_t(self.InSampleRate), 0)
	C.av_opt_set_int(avr, C.CString("out_sample_rate"), C.int64_t(self.OutSampleRate), 0)
	C.av_opt_set_int(avr, C.CString("in_sample_fmt"), C.int64_t(sampleFormatAV2FF(self.InSampleFormat)), 0)
	C.av_opt_set_int(avr, C.CString("out_sample_fmt"), C.int64_t(sampleFormatAV2FF(self.OutSampleFormat)), 0)
	self.avr = avr
	runtime.SetFinalizer(self, freeResampler)
	if C.avresample_open(avr) != 0 {
		err = fmt.Errorf("avresample_open failed")
		return
	}
	self.inframe = C.av_frame_alloc()
	self.outframe = C.av_frame_alloc()
	return
}

func (self *Resampler) Resample(in av.AudioFrame) (out av.AudioFrame) {
	return
}

type AudioEncoder struct {
	ff *ffctx
	SampleRate int
	BitRate int
	ChannelLayout av.ChannelLayout
	SampleFormat av.SampleFormat
	FrameSampleCount int
	framebuf av.AudioFrame
	codecData av.AudioCodecData
	resampler *Resampler
}

func sampleFormatAV2FF(sampleFormat av.SampleFormat) (ffsamplefmt C.int) {
	switch sampleFormat {
	case av.U8:
		ffsamplefmt = C.AV_SAMPLE_FMT_U8
	case av.S16:
		ffsamplefmt = C.AV_SAMPLE_FMT_S16
	case av.S32:
		ffsamplefmt = C.AV_SAMPLE_FMT_S32
	case av.FLT:
		ffsamplefmt = C.AV_SAMPLE_FMT_FLT
	case av.DBL:
		ffsamplefmt = C.AV_SAMPLE_FMT_DBL
	case av.U8P:
		ffsamplefmt = C.AV_SAMPLE_FMT_U8P
	case av.S16P:
		ffsamplefmt = C.AV_SAMPLE_FMT_S16P
	case av.S32P:
		ffsamplefmt = C.AV_SAMPLE_FMT_S32P
	case av.FLTP:
		ffsamplefmt = C.AV_SAMPLE_FMT_FLTP
	case av.DBLP:
		ffsamplefmt = C.AV_SAMPLE_FMT_DBLP
	}
	return
}

func sampleFormatFF2AV(ffsamplefmt int32) (sampleFormat av.SampleFormat) {
	switch ffsamplefmt {
	case C.AV_SAMPLE_FMT_U8:          ///< unsigned 8 bits
		sampleFormat = av.U8
	case C.AV_SAMPLE_FMT_S16:         ///< signed 16 bits
		sampleFormat = av.S16
	case C.AV_SAMPLE_FMT_S32:         ///< signed 32 bits
		sampleFormat = av.S32
	case C.AV_SAMPLE_FMT_FLT:         ///< float
		sampleFormat = av.FLT
	case C.AV_SAMPLE_FMT_DBL:         ///< double
		sampleFormat = av.DBL
	case C.AV_SAMPLE_FMT_U8P:         ///< unsigned 8 bits, planar
		sampleFormat = av.U8P
	case C.AV_SAMPLE_FMT_S16P:        ///< signed 16 bits, planar
		sampleFormat = av.S16P
	case C.AV_SAMPLE_FMT_S32P:        ///< signed 32 bits, planar
		sampleFormat = av.S32P
	case C.AV_SAMPLE_FMT_FLTP:        ///< float, planar
		sampleFormat = av.FLTP
	case C.AV_SAMPLE_FMT_DBLP:        ///< double, planar
		sampleFormat = av.DBLP
	}
	return
}

func (self *AudioEncoder) Setup() (err error) {
	ff := &self.ff.ff

	ff.frame = C.av_frame_alloc()
	if self.SampleFormat == av.SampleFormat(0) {
		self.SampleFormat = av.FLTP
	}
	if self.BitRate == 0 {
		self.BitRate = 50000
	}
	if self.SampleRate == 0 {
		self.SampleRate = 44100
	}
	if self.ChannelLayout == av.ChannelLayout(0) {
		self.ChannelLayout = av.CH_STEREO
	}

	C.set_sample_fmt(ff.codecCtx, C.int(self.SampleFormat))
	ff.codecCtx.sample_rate = C.int(self.SampleRate)
	ff.codecCtx.bit_rate = C.int(self.BitRate)
	ff.codecCtx.channel_layout = channelLayoutAV2FF(self.ChannelLayout)
	ff.codecCtx.strict_std_compliance = C.FF_COMPLIANCE_EXPERIMENTAL
	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("avcodec_open2 failed")
		return
	}
	self.SampleFormat = sampleFormatFF2AV(ff.codecCtx.sample_fmt)
	self.FrameSampleCount = int(ff.codecCtx.frame_size)

	extradata := C.GoBytes(unsafe.Pointer(ff.codecCtx.extradata), ff.codecCtx.extradata_size)
	switch ff.codecCtx.codec_id {
	case C.AV_CODEC_ID_AAC:
		if self.codecData, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(extradata); err != nil {
			return
		}

	default:
		self.codecData = AudioCodecData{
			channelLayout: self.ChannelLayout,
			sampleFormat: self.SampleFormat,
			sampleRate: self.SampleRate,
			codecId: ff.codecCtx.codec_id,
			extradata: extradata,
		}
	}

	return
}

func (self *AudioEncoder) CodecData() (codec av.AudioCodecData) {
	return self.codecData
}

func (self *AudioEncoder) Encode(frame av.AudioFrame) (gotPkt bool, pkt av.Packet, err error) {
	ff := &self.ff.ff

	if self.FrameSampleCount != 0 {
		self.framebuf = self.framebuf.Concat(frame)
		if self.framebuf.SampleCount < self.FrameSampleCount {
			return
		}
		frame = self.framebuf.Slice(0, self.FrameSampleCount)
		self.framebuf = self.framebuf.Slice(self.FrameSampleCount, self.framebuf.SampleCount)
	}

	cpkt := C.AVPacket{}
	cgotpkt := C.int(0)
	audioFrameAssignToFF(frame, ff.frame)
	cerr := C.avcodec_encode_audio2(ff.codecCtx, &cpkt, ff.frame, &cgotpkt)
	if cerr < C.int(0) {
		err = fmt.Errorf("avcodec_encode_audio2 failed: %d", cerr)
		return
	}

	if cgotpkt != 0 {
		gotPkt = true
		pkt.Data = C.GoBytes(unsafe.Pointer(cpkt.data), cpkt.size)
		pkt.Duration = float64(frame.SampleCount)/float64(self.SampleRate)
		C.av_free_packet(&cpkt)
	}

	return
}

func (self *AudioEncoder) Close() {
	freeFFCtx(self.ff)
}

func audioFrameAssignToAV(f *C.AVFrame, frame *av.AudioFrame) {
	frame.SampleCount = int(f.nb_samples)
	frame.SampleFormat = sampleFormatFF2AV(int32(f.format))
	frame.ChannelLayout = channelLayoutFF2AV(f.channel_layout)
	channels := int(f.channels)
	frame.Data = make([][]byte, channels)
	for i := 0; i < channels; i++ {
		frame.Data[i] = C.GoBytes(unsafe.Pointer(f.data[i]), f.linesize[i])
	}
}

func audioFrameAssignToFF(frame av.AudioFrame, f *C.AVFrame) {
	f.nb_samples = C.int(frame.SampleCount)
	f.format = C.int(sampleFormatAV2FF(frame.SampleFormat))
	f.channel_layout = channelLayoutAV2FF(frame.ChannelLayout)
	for i := range frame.Data {
		f.data[i] = (*C.uint8_t)(unsafe.Pointer(&frame.Data[i]))
		f.linesize[i] = C.int(len(frame.Data[i]))
	}
}

func channelLayoutFF2AV(layout C.uint64_t) (channelLayout av.ChannelLayout) {
	if layout & C.AV_CH_FRONT_CENTER != 0 {
		channelLayout |= av.CH_FRONT_CENTER
	}
	if layout & C.AV_CH_FRONT_LEFT != 0 {
		channelLayout |= av.CH_FRONT_LEFT
	}
	if layout & C.AV_CH_FRONT_RIGHT != 0 {
		channelLayout |= av.CH_FRONT_RIGHT
	}
	if layout & C.AV_CH_BACK_CENTER != 0 {
		channelLayout |= av.CH_BACK_CENTER
	}
	if layout & C.AV_CH_BACK_LEFT != 0 {
		channelLayout |= av.CH_BACK_LEFT
	}
	if layout & C.AV_CH_BACK_RIGHT != 0 {
		channelLayout |= av.CH_BACK_RIGHT
	}
	if layout & C.AV_CH_SIDE_LEFT != 0 {
		channelLayout |= av.CH_SIDE_LEFT
	}
	if layout & C.AV_CH_SIDE_RIGHT != 0 {
		channelLayout |= av.CH_SIDE_RIGHT
	}
	if layout & C.AV_CH_LOW_FREQUENCY != 0 {
		channelLayout |= av.CH_LOW_FREQ
	}
	return
}

func channelLayoutAV2FF(channelLayout av.ChannelLayout) (layout C.uint64_t) {
	if channelLayout & av.CH_FRONT_CENTER != 0 {
		layout |= C.AV_CH_FRONT_CENTER
	}
	if channelLayout & av.CH_FRONT_LEFT != 0 {
		layout |= C.AV_CH_FRONT_LEFT
	}
	if channelLayout & av.CH_FRONT_RIGHT != 0 {
		layout |= C.AV_CH_FRONT_RIGHT
	}
	if channelLayout & av.CH_BACK_CENTER != 0 {
		layout |= C.AV_CH_BACK_CENTER
	}
	if channelLayout & av.CH_BACK_LEFT != 0 {
		layout |= C.AV_CH_BACK_LEFT
	}
	if channelLayout & av.CH_BACK_RIGHT != 0 {
		layout |= C.AV_CH_BACK_RIGHT
	}
	if channelLayout & av.CH_SIDE_LEFT != 0 {
		layout |= C.AV_CH_SIDE_LEFT
	}
	if channelLayout & av.CH_SIDE_RIGHT != 0 {
		layout |= C.AV_CH_SIDE_RIGHT
	}
	if channelLayout & av.CH_LOW_FREQ != 0 {
		layout |= C.AV_CH_LOW_FREQUENCY
	}
	return
}

type AudioDecoder struct {
	ff *ffctx
	ChannelLayout av.ChannelLayout
	SampleFormat av.SampleFormat
	Extradata []byte
}

func (self *AudioDecoder) Setup() (err error) {
	ff := &self.ff.ff

	ff.frame = C.av_frame_alloc()

	if len(self.Extradata) > 0 {
		ff.codecCtx.extradata = (*C.uint8_t)(unsafe.Pointer(&self.Extradata[0]))
		ff.codecCtx.extradata_size = C.int(len(self.Extradata))
	}

	ff.codecCtx.channel_layout = channelLayoutAV2FF(self.ChannelLayout)
	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("avcodec_open2 failed")
		return
	}
	self.SampleFormat = sampleFormatFF2AV(ff.codecCtx.sample_fmt)
	self.ChannelLayout = channelLayoutFF2AV(ff.codecCtx.channel_layout)

	return
}

func (self *AudioDecoder) Decode(data []byte) (gotFrame bool, frame av.AudioFrame, err error) {
	ff := &self.ff.ff

	cgotpkt := C.int(0)
	cerr := C.wrap_avcodec_decode_audio4(ff.codecCtx, ff.frame, unsafe.Pointer(&data[0]), C.int(len(data)), &cgotpkt)
	if cerr < C.int(0) {
		err = fmt.Errorf("avcodec_decode_audio4 failed: %d", cerr)
		return
	}

	if cgotpkt != C.int(0) {
		gotFrame = true
		audioFrameAssignToAV(ff.frame, &frame)
	}

	return
}

func (self *AudioDecoder) Close() {
	freeFFCtx(self.ff)
}

func HasEncoder(name string) bool {
	return C.avcodec_find_encoder_by_name(C.CString(name)) != nil
}

func HasDecoder(name string) bool {
	return C.avcodec_find_decoder_by_name(C.CString(name)) != nil
}

//func EncodersList() []string
//func DecodersList() []string

func newFFCtxByCodec(codec *C.AVCodec) (ff *ffctx, err error) {
	ff = &ffctx{}
	ff.ff.codec = codec
	ff.ff.codecCtx = C.avcodec_alloc_context3(codec)
	runtime.SetFinalizer(ff, freeFFCtx)
	return
}

func freeFFCtx(self *ffctx) {
	ff := &self.ff
	if ff.frame != nil {
		C.av_frame_free(&ff.frame)
		ff.frame = nil
	}
	if ff.codecCtx != nil {
		C.avcodec_close(ff.codecCtx)
		C.av_free(ff.codecCtx)
		ff.codecCtx = nil
	}
}

func NewAudioEncoder(
	name string,
	sampleFormat av.SampleFormat, sampleRate int, channelLayout av.ChannelLayout, bitRate int,
) (enc *AudioEncoder, err error) {
	_enc := &AudioEncoder{}

	codec := C.avcodec_find_encoder_by_name(C.CString(name))
	if codec == nil {
		err = fmt.Errorf("cannot find encoder=%s", name)
		return
	}
	if C.avcodec_get_type(codec.id) != C.AVMEDIA_TYPE_AUDIO {
		err = fmt.Errorf("encoder=%s type is not audio", name)
		return
	}

	if _enc.ff, err = newFFCtxByCodec(codec); err != nil {
		return
	}
	_enc.SampleFormat = sampleFormat
	_enc.SampleRate = sampleRate
	_enc.ChannelLayout = channelLayout
	_enc.BitRate = bitRate
	if err = _enc.Setup(); err != nil {
		return
	}
	enc = _enc
	return
}

func NewAudioDecoder(codec av.AudioCodecData) (dec *AudioDecoder, err error) {
	_dec := &AudioDecoder{}
	var id uint32

	switch codec.Type() {
	case av.AAC:
		if aaccodec, ok := codec.(av.AACCodecData); ok {
			_dec.Extradata = aaccodec.MPEG4AudioConfigBytes()
			id = C.AV_CODEC_ID_AAC
		} else {
			err = fmt.Errorf("aac CodecData must be av.AACCodecData")
			return
		}

	default:
		if ffcodec, ok := codec.(AudioCodecData); ok {
			_dec.SampleFormat = ffcodec.sampleFormat
			_dec.ChannelLayout = ffcodec.channelLayout
			_dec.Extradata = ffcodec.extradata
			id = ffcodec.codecId
		} else {
			err = fmt.Errorf("invalid CodecData for ffmpeg to decode")
			return
		}
	}

	c := C.avcodec_find_decoder(id)
	if c == nil {
		err = fmt.Errorf("cannot find decoder id=%d", id)
		return
	}

	if C.avcodec_get_type(c.id) != C.AVMEDIA_TYPE_AUDIO {
		err = fmt.Errorf("decoder id=%d type is not audio", c.id)
		return
	}

	if _dec.ff, err = newFFCtxByCodec(c); err != nil {
		return
	}
	if err = _dec.Setup(); err != nil {
		return
	}

	dec = _dec
	return
}

type AudioCodecData struct {
	codecId uint32
	sampleFormat av.SampleFormat
	channelLayout av.ChannelLayout
	sampleRate int
	extradata []byte
}

func (self AudioCodecData) Type() int {
	return int(self.codecId)
}

func (self AudioCodecData) IsAudio() bool {
	return true
}

func (self AudioCodecData) IsVideo() bool {
	return false
}

func (self AudioCodecData) SampleRate() int {
	return self.sampleRate
}

func (self AudioCodecData) SampleFormat() av.SampleFormat {
	return self.sampleFormat
}

func (self AudioCodecData) ChannelLayout() av.ChannelLayout {
	return self.channelLayout
}

