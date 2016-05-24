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
				int samplerate; int bitrate;
				int channels;
			} aacenc_t ;

			static int aacenc_new(aacenc_t *m) {
				m->c = avcodec_find_encoder_by_name("aac");
				m->ctx = avcodec_alloc_context3(m->c);
				m->ctx->sample_fmt = AV_SAMPLE_FMT_FLTP;
				m->ctx->sample_rate = m->samplerate;
				m->ctx->bit_rate = m->bitrate;
				m->ctx->channels = m->channels;
		  	m->ctx->strict_std_compliance = FF_COMPLIANCE_EXPERIMENTAL;
				m->f = av_frame_alloc();
				int r = avcodec_open2(m->ctx, m->c, 0);
				//av_log(m->ctx, AV_LOG_DEBUG, "extra %d\n", m->ctx->extradata_size);
				return r;
			}

			static void aacenc_encode(aacenc_t *m) {
				AVPacket pkt;
				av_init_packet(&pkt);
				pkt.data = m->buf;
				pkt.size = sizeof(m->buf);
				m->f->nb_samples = 1024;
				m->f->extended_data = m->f->data;
				m->f->linesize[0] = 1024*4;
				//m->f->linesize[1] = 1024*4;
				avcodec_encode_audio2(m->ctx, &pkt, m->f, &m->got);
				av_log(m->ctx, AV_LOG_DEBUG, "got %d size %d\n", m->got, pkt.size);
				m->size = pkt.size;
			}
	*/
	"C"
	"errors"
	"unsafe"
)

type AACEncoder struct {
	m      C.aacenc_t
	Header []byte
}

// only supported fltp,stereo,44100HZ. If you need other config, it's easy to modify code
func NewAACEncoder() (m *AACEncoder, err error) {
	m = &AACEncoder{}
	m.m.samplerate = 44100
	m.m.bitrate = 50000
	m.m.channels = 1
	r := C.aacenc_new(&m.m)
	if int(r) != 0 {
		err = errors.New("open codec failed")
		return
	}
	m.Header = make([]byte, (int)(m.m.ctx.extradata_size))
	C.memcpy(
		unsafe.Pointer(&m.Header[0]),
		unsafe.Pointer(m.m.ctx.extradata),
		(C.size_t)(len(m.Header)),
	)
	return
}

func (m *AACEncoder) Encode(sample []byte) (ret []byte, err error) {
	m.m.f.data[0] = (*C.uint8_t)(unsafe.Pointer(&sample[0]))
	//m.m.f.data[1] = (*C.uint8_t)(unsafe.Pointer(&sample[1024*4]))
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

