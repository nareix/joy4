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

type AudioEncoder struct {
	ff *ffctx
	SampleRate int
	BitRate int
	ChannelCount int
	SampleFormat av.SampleFormat
	FrameSampleCount int
	codecData av.AudioCodecData
}

func convffSampleFormat(ffsamplefmt int32) (sampleFormat av.SampleFormat, err error) {
	switch ffsamplefmt {
	case C.AV_SAMPLE_FMT_U8:          ///< unsigned 8 bits
		sampleFormat = av.U8
	case C.AV_SAMPLE_FMT_S16:         ///< signed 16 bits
		sampleFormat = av.S16
	case C.AV_SAMPLE_FMT_S32:         ///< signed 32 bits
		sampleFormat = av.U32
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
	default:
		err = fmt.Errorf("ffsamplefmt=%d invalid", ffsamplefmt)
		return
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
	if self.ChannelCount == 0 {
		self.ChannelCount = 2
	}
	C.set_sample_fmt(ff.codecCtx, C.int(self.SampleFormat))
	ff.codecCtx.sample_rate = C.int(self.SampleRate)
	ff.codecCtx.bit_rate = C.int(self.BitRate)
	ff.codecCtx.channels = C.int(self.ChannelCount)
	ff.codecCtx.strict_std_compliance = C.FF_COMPLIANCE_EXPERIMENTAL
	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("avcodec_open2 failed")
		return
	}
	if self.SampleFormat, err = convffSampleFormat(ff.codecCtx.sample_fmt); err != nil {
		return
	}
	self.ChannelCount = int(ff.codecCtx.channels)
	self.FrameSampleCount = int(ff.codecCtx.frame_size)

	extradata := C.GoBytes(unsafe.Pointer(ff.codecCtx.extradata), ff.codecCtx.extradata_size)
	switch ff.codecCtx.codec_id {
	case C.AV_CODEC_ID_AAC:
		if self.codecData, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(extradata); err != nil {
			return
		}

	default:
		self.codecData = AudioCodecData{
			channelCount: self.ChannelCount,
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

func (self *AudioEncoder) Encode(sample []byte, flush bool) (gotPkt bool, pkt []byte, err error) {
	ff := &self.ff.ff
	nbSamples := self.FrameSampleCount
	channelCount := int(ff.codecCtx.channels)
	sampleSize := int(C.av_get_bytes_per_sample(ff.codecCtx.sample_fmt))
	expectedSize := nbSamples*sampleSize*channelCount

	frame := ff.frame
	if flush {
		frame = nil
	} else {
		if len(sample) != expectedSize {
			err = fmt.Errorf("len(sample) should be %d", expectedSize)
			return
		}

		frame.nb_samples = C.int(nbSamples)
		frame.format = C.int(ff.codecCtx.sample_fmt)
		frame.channel_layout = ff.codecCtx.channel_layout
		if C.av_sample_fmt_is_planar(ff.codecCtx.sample_fmt) != 0 {
			for i := 0; i < self.ChannelCount; i++ {
				frame.data[i] = (*C.uint8_t)(unsafe.Pointer(&sample[i*nbSamples*sampleSize]))
				frame.linesize[i] = C.int(nbSamples*sampleSize)
			}
		} else {
			frame.data[0] = (*C.uint8_t)(unsafe.Pointer(&sample[0]))
			frame.linesize[0] = C.int(channelCount*nbSamples*sampleSize)
		}
		//frame.extended_data = &frame.data[0]
	}

	cpkt := C.AVPacket{}
	cgotpkt := C.int(0)
	cerr := C.avcodec_encode_audio2(ff.codecCtx, &cpkt, frame, &cgotpkt)
	if cerr < C.int(0) {
		err = fmt.Errorf("avcodec_encode_audio2 failed: %d", cerr)
		return
	}

	if cgotpkt != 0 {
		gotPkt = true
		pkt = C.GoBytes(unsafe.Pointer(cpkt.data), cpkt.size)
		C.av_free_packet(&cpkt)
	}

	return
}

func (self *AudioEncoder) Close() {
	freeFFCtx(self.ff)
}

type AudioDecoder struct {
	ff *ffctx
	ChannelCount int
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

	ff.codecCtx.channels = C.int(self.ChannelCount)
	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("avcodec_open2 failed")
		return
	}
	if self.SampleFormat, err = convffSampleFormat(ff.codecCtx.sample_fmt); err != nil {
		return
	}
	self.ChannelCount = int(ff.codecCtx.channels)

	return
}

func (self *AudioDecoder) Decode(frame []byte) (gotPkt bool, pkt []byte, err error) {
	ff := &self.ff.ff

	cgotpkt := C.int(0)
	cerr := C.wrap_avcodec_decode_audio4(ff.codecCtx, ff.frame, unsafe.Pointer(&frame[0]), C.int(len(frame)), &cgotpkt)
	if cerr < C.int(0) {
		err = fmt.Errorf("avcodec_decode_audio4 failed: %d", cerr)
		return
	}

	if cgotpkt != C.int(0) {
		gotPkt = true
		//pkt = C.GoBytes(unsafe.Pointer(cpkt.data), cpkt.size)
		size := C.av_samples_get_buffer_size(nil, ff.codecCtx.channels, ff.frame.nb_samples, ff.codecCtx.sample_fmt, C.int(1))
		pkt = C.GoBytes(unsafe.Pointer(ff.frame.data[0]), size)
	}

	return
}

func (self *AudioDecoder) Close() {
	freeFFCtx(self.ff)
}

//func HasEncoder(name string) bool
//func HasDecoder(name string) bool
//func EncodersList() []string
//func DecodersList() []string

func newFFCtxByCodec(codec *C.AVCodec) (ff *ffctx, err error) {
	if codec == nil {
		err = fmt.Errorf("AVCodec not found")
		return
	}
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
	sampleFormat av.SampleFormat, sampleRate int, channelCount int, bitRate int,
) (enc *AudioEncoder, err error) {
	_enc := &AudioEncoder{}
	if _enc.ff, err = newFFCtxByCodec(C.avcodec_find_encoder_by_name(C.CString(name))); err != nil {
		return
	}
	_enc.SampleFormat = sampleFormat
	_enc.SampleRate = sampleRate
	_enc.ChannelCount = channelCount
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
			_dec.ChannelCount = ffcodec.channelCount
			_dec.Extradata = ffcodec.extradata
			id = ffcodec.codecId
		} else {
			err = fmt.Errorf("invalid CodecData for ffmpeg to decode")
			return
		}
	}

	if _dec.ff, err = newFFCtxByCodec(C.avcodec_find_decoder(id)); err != nil {
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
	channelCount int
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

func (self AudioCodecData) ChannelCount() int {
	return self.channelCount
}

