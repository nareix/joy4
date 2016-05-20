package codec

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
	int wrap_av_get_bytes_per_sample(int sample_fmt) {
		return av_get_bytes_per_sample(sample_fmt);
	}
	*/
	"C"
	"unsafe"
	"fmt"
)

type SampleFormat int

func (self SampleFormat) BytesPerSample() int {
	return int(C.wrap_av_get_bytes_per_sample(C.int(self)))
}

const (
	S16 = SampleFormat(C.AV_SAMPLE_FMT_S16)
	FLTP = SampleFormat(C.AV_SAMPLE_FMT_FLTP)
)

type AudioEncoder struct {
	ff C.FFCtx
	SampleRate int
	BitRate int
	ChannelCount int
	SampleFormat SampleFormat
	FrameSampleCount int
}

func (self *AudioEncoder) Setup() (err error) {
	ff := &self.ff

	ff.frame = C.av_frame_alloc()
	if self.BitRate == 0 {
		self.BitRate = 50000
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
	self.SampleFormat = SampleFormat(int(ff.codecCtx.sample_fmt))
	self.ChannelCount = int(ff.codecCtx.channels)
	self.FrameSampleCount = int(ff.codecCtx.frame_size)

	return
}

func (self *AudioEncoder) Extradata() (data []byte) {
	data = C.GoBytes(unsafe.Pointer(self.ff.codecCtx.extradata), self.ff.codecCtx.extradata_size)
	return
}

func (self *AudioEncoder) Encode(sample []byte, flush bool) (gotPkt bool, pkt []byte, err error) {
	ff := &self.ff
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
	cerr := C.avcodec_encode_audio2(self.ff.codecCtx, &cpkt, frame, &cgotpkt)
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

type AudioDecoder struct {
	ff C.FFCtx
	ChannelCount int
	SampleFormat SampleFormat
	Extradata []byte
}

func (self *AudioDecoder) Setup() (err error) {
	ff := &self.ff

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
	self.SampleFormat = SampleFormat(int(ff.codecCtx.sample_fmt))
	self.ChannelCount = int(ff.codecCtx.channels)

	return
}

func (self *AudioDecoder) Decode(frame []byte) (gotPkt bool, pkt []byte, err error) {
	ff := &self.ff

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

func FindAudioEncoderByName(name string) (enc *AudioEncoder) {
	ff := C.FFCtx{}
	ff.codec = C.avcodec_find_encoder_by_name(C.CString(name))
	if ff.codec != nil {
		ff.codecCtx = C.avcodec_alloc_context3(ff.codec)
		if ff.codecCtx != nil {
			return &AudioEncoder{ff: ff}
		}
	}
	return nil
}

func FindAudioDecoderByName(name string) (dec *AudioDecoder) {
	ff := C.FFCtx{}
	ff.codec = C.avcodec_find_decoder_by_name(C.CString(name))
	if ff.codec != nil {
		ff.codecCtx = C.avcodec_alloc_context3(ff.codec)
		if ff.codecCtx != nil {
			return &AudioDecoder{ff: ff}
		}
	}
	return nil
}

