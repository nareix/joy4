
package codec

import (
	/*
	#include <libavcodec/avcodec.h>
	#include <libavutil/avutil.h>
	#include <string.h>

	typedef struct {
		AVCodec *c;
		AVCodecContext *ctx;
		AVFrame *f;
		int got;
		uint8_t buf[1024*10]; int size;
	} aacenc_t ;

	static void aacenc_new(aacenc_t *m) {
		m->c = avcodec_find_encoder(CODEC_ID_AAC);
		m->ctx = avcodec_alloc_context3(m->c);
		m->ctx->sample_fmt = AV_SAMPLE_FMT_FLTP;
		m->ctx->sample_rate = 44100;
		m->ctx->bit_rate = 100000;
		m->ctx->channels = 2;
  	m->ctx->strict_std_compliance = FF_COMPLIANCE_EXPERIMENTAL;
		m->f = avcodec_alloc_frame();
		avcodec_open2(m->ctx, m->c, 0);
		av_log(m->ctx, AV_LOG_DEBUG, "extra %d\n", m->ctx->extradata_size);
	}

	static void aacenc_encode(aacenc_t *m) {
		AVPacket pkt;
		av_init_packet(&pkt);
		pkt.data = m->buf;
		pkt.size = sizeof(m->buf);
		m->f->nb_samples = 1024;
		m->f->extended_data = m->f->data;
		m->f->linesize[0] = 4096;
		avcodec_encode_audio2(m->ctx, &pkt, m->f, &m->got);
		av_log(m->ctx, AV_LOG_DEBUG, "got %d size %d\n", m->got, pkt.size);
		m->size = pkt.size;
	}
	*/
	"C"
	"unsafe"
	"errors"
)

type AACEncoder struct {
	m C.aacenc_t
	Cfg []byte // AAC Audio config
}

// only supported fltp,stereo,44100khz. If you need other config, it's easy to modify code
func NewAACEncoder() (m *AACEncoder) {
	m = &AACEncoder{}
	C.aacenc_new(&m.m)
	m.Cfg = make([]byte, (int)(m.m.ctx.extradata_size))
	C.memcpy(
		unsafe.Pointer(&m.Cfg[0]),
		unsafe.Pointer(&m.m.ctx.extradata),
		(C.size_t)(len(m.Cfg)),
	)
	return
}

func (m *AACEncoder) Encode(sample []byte) (ret []byte, err error) {
	m.m.f.data[0] = (*C.uint8_t)(unsafe.Pointer(&sample[0]))
	m.m.f.data[1] = (*C.uint8_t)(unsafe.Pointer(&sample[4096]))
	C.aacenc_encode(&m.m)
	if int(m.m.got) == 0 {
		err = errors.New("no data")
		return
	}
	ret = make([]byte, (int)(m.m.size))
	C.memcpy(
		unsafe.Pointer(&ret[0]),
		unsafe.Pointer(&m.m.buf[0]),
		(C.size_t)(m.m.size),
	)
	return
}

