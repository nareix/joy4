
package codec

import (
	/*
	#include <libavcodec/avcodec.h>
	#include <libavutil/avutil.h>
	#include <string.h>
	#include <stdio.h>

	typedef struct {
		AVCodec *c;
		AVCodecContext *ctx;
		AVFrame *f;
		int got;
	} aacdec_t ;

	static int aacdec_new(aacdec_t *m, uint8_t *buf, int len) {
		m->c = avcodec_find_decoder(CODEC_ID_AAC);
		m->ctx = avcodec_alloc_context3(m->c);
		m->f = avcodec_alloc_frame();
		m->ctx->extradata = buf;
		m->ctx->extradata_size = len;
		m->ctx->debug = 0x3;
		av_log(m->ctx, AV_LOG_DEBUG, "m %p\n", m);
		return avcodec_open2(m->ctx, m->c, 0);
	}

	static void aacdec_decode(aacdec_t *m, uint8_t *data, int len) {
		AVPacket pkt;
		av_init_packet(&pkt);
		pkt.data = data;
		pkt.size = len;
		av_log(m->ctx, AV_LOG_DEBUG, "decode %p\n", m);
		avcodec_decode_audio4(m->ctx, m->f, &m->got, &pkt);
		av_log(m->ctx, AV_LOG_DEBUG, "got %d\n", m->got);
	}
	*/
	"C"
	"unsafe"
	"errors"
)

type AACDecoder struct {
	m C.aacdec_t
}

func NewAACDecoder(cfg []byte) (m *AACDecoder, err error) {
	m = &AACDecoder{}
	r := C.aacdec_new(
		&m.m,
		(*C.uint8_t)(unsafe.Pointer(&cfg[0])),
		(C.int)(len(cfg)),
	)
	if int(r) != 0 {
		err = errors.New("avcodec open failed")
	}
	return
}

func (m *AACDecoder) Decode(data []byte) (sample []byte, err error) {
	C.aacdec_decode(
		&m.m,
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		(C.int)(len(data)),
	)
	if int(m.m.got) == 0 {
		err = errors.New("no data")
		return
	}
	sample = make([]byte, 8192)
	for i := 0; i < 2; i++ {
		C.memcpy(
			unsafe.Pointer(&sample[i*4096]),
			unsafe.Pointer(m.m.f.data[i]),
			(C.size_t)(4096),
		)
	}
	return
}

