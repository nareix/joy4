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

		static int aacdec_decode(aacdec_t *m, uint8_t *data, int len) {
			AVPacket pkt;
			av_init_packet(&pkt);
			pkt.data = data;
			pkt.size = len;
			av_log(m->ctx, AV_LOG_DEBUG, "decode %p\n", m);
			return avcodec_decode_audio4(m->ctx, m->f, &m->got, &pkt);
		}
	*/
	"C"
	"errors"
	"unsafe"
)

type AACDecoder struct {
	m C.aacdec_t
}

func NewAACDecoder(header []byte) (m *AACDecoder, err error) {
	m = &AACDecoder{}
	r := C.aacdec_new(&m.m,
		(*C.uint8_t)(unsafe.Pointer(&header[0])),
		(C.int)(len(header)),
	)
	if int(r) < 0 {
		err = errors.New("open codec failed")
	}
	return
}

func (m *AACDecoder) Decode(data []byte) (sample []byte, err error) {
	r := C.aacdec_decode(
		&m.m,
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		(C.int)(len(data)),
	)
	if int(r) < 0 {
		err = errors.New("decode failed")
		return
	}
	if int(m.m.got) == 0 {
		err = errors.New("no data")
		return
	}
	size := int(m.m.f.linesize[0]) * 2
	sample = make([]byte, size*2)
	for i := 0; i < 2; i++ {
		C.memcpy(
			unsafe.Pointer(&sample[i*size]),
			unsafe.Pointer(m.m.f.data[i]),
			(C.size_t)(size),
		)
	}
	return
}
