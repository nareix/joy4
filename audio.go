package codec

import (
	// #include "ffmpeg.h"
	"C"
	"unsafe"
	"fmt"
)

const (
	S16 = iota+1
	FLTP
)

type AudioEncoder struct {
	ff C.FFCtx
	SampleRate int
	BitRate int
	ChannelCount int
	SampleFormat int
	sampleSize int
}

func (self *AudioEncoder) Setup() (err error) {
	ff := &self.ff

	switch self.SampleFormat {
	case S16:
		ff.codecCtx.sample_fmt = C.AV_SAMPLE_FMT_S16
		self.sampleSize = 2
	case FLTP:
		ff.codecCtx.sample_fmt = C.AV_SAMPLE_FMT_FLTP
		self.sampleSize = 4
	default:
		err = fmt.Errorf("unsupported sample format")
		return
	}

	if self.BitRate == 0 {
		self.BitRate = 50000
	}

	ff.frame = C.av_frame_alloc()
	ff.codecCtx.sample_rate = C.int(self.SampleRate)
	ff.codecCtx.bit_rate = C.int(self.BitRate)
	ff.codecCtx.channels = C.int(self.ChannelCount)
	ff.codecCtx.strict_std_compliance = C.FF_COMPLIANCE_EXPERIMENTAL
	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("avcodec_open2 failed")
		return
	}

	return
}

func (self *AudioEncoder) Extradata() (data []byte) {
	data = make([]byte, (int)(self.ff.codecCtx.extradata_size))
	C.memcpy(
		unsafe.Pointer(&data[0]),
		unsafe.Pointer(self.ff.codecCtx.extradata),
		(C.size_t)(len(data)),
	)
	return
}

func (self *AudioEncoder) Encode(sample []byte, flush bool) (gotPkt bool, pkt []byte, err error) {
	nbSamples := 1024
	expectedSize := nbSamples*self.sampleSize*self.ChannelCount

	if len(sample) != expectedSize {
		err = fmt.Errorf("len(sample) should be %d", expectedSize)
		return
	}

	frame := self.ff.frame
	frame.nb_samples = C.int(nbSamples)
	for i := 0; i < self.ChannelCount; i++ {
		frame.data[i] = (*C.uint8_t)(unsafe.Pointer(&sample[i*nbSamples*self.sampleSize]))
		frame.linesize[i] = C.int(nbSamples*self.sampleSize)
	}
	frame.extended_data = &frame.data[0]

	cpkt := C.AVPacket{}
	cgotpkt := C.int(0)
	cerr := C.avcodec_encode_audio2(self.ff.codecCtx, &cpkt, frame, &cgotpkt)
	if cerr < C.int(0) {
		err = fmt.Errorf("avcodec_encode_audio2 failed: %d", cerr)
		return
	}

	if cgotpkt != 0 {
		gotPkt = true
		pkt = make([]byte, (int)(cpkt.size))
		C.memcpy(
			unsafe.Pointer(&pkt[0]),
			unsafe.Pointer(cpkt.data),
			(C.size_t)(len(pkt)),
		)
	}

	return
}

type AudioDecoder struct {
	ff C.FFCtx
	Extradata []byte
}

func (self *AudioDecoder) Setup() (err error) {
	ff := &self.ff

	ff.frame = C.av_frame_alloc()

	if len(self.Extradata) > 0 {
		ff.codecCtx.extradata = (*C.uint8_t)(unsafe.Pointer(&self.Extradata[0]))
		ff.codecCtx.extradata_size = C.int(len(self.Extradata))
	}

	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("avcodec_open2 failed")
		return
	}

	return
}

func (self *AudioDecoder) Decode(frame []byte) (gotPkt bool, pkt []byte, err error) {
	ff := &self.ff

	cpkt := C.AVPacket{
		data: (*C.uint8_t)(unsafe.Pointer(&frame[0])),
		size: C.int(len(frame)),
	}
	cgotpkt := C.int(0)
	cerr := C.avcodec_decode_audio4(ff.codecCtx, ff.frame, &cgotpkt, &cpkt);
	if cerr < C.int(0) {
		err = fmt.Errorf("avcodec_decode_audio4 failed: %d", cerr)
		return
	}

	if cgotpkt != C.int(0) {
		gotPkt = true
		pkt = make([]byte, (int)(cpkt.size))
		C.memcpy(
			unsafe.Pointer(&pkt[0]),
			unsafe.Pointer(cpkt.data),
			(C.size_t)(len(pkt)),
		)
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

