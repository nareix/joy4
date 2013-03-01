
package codec

import (

/*
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <x264.h>

typedef struct {
	x264_t *x;
	int w, h;
	uint8_t *pps; int ppslen;
	uint8_t *sei; int seilen;
	x264_picture_t in;
	x264_nal_t *nal; int nnal; int nallen;
} h264enc_t;

static void h264enc_new(h264enc_t *m) {
	x264_param_t p;

	x264_param_default(&p);
	x264_param_default_preset(&p, "ultrafast", "zerolatency");
	x264_param_apply_profile(&p, "main");

	p.rc.i_bitrate = 50;
	p.rc.i_rc_method = X264_RC_ABR;
	p.i_width = m->w;
	p.i_height = m->h;
	p.i_csp = X264_CSP_I420;

	m->x = x264_encoder_open(&p);
	if (!m->x) {
		return ;
	}
	
	x264_nal_t *nal;
	int nnal, i;

	uint8_t *pps;
	m->ppslen = x264_encoder_headers(m->x, &nal, &nnal);
	m->pps = pps = malloc(m->ppslen);

	for (i = 0; i < nnal; i++) {
		//printf("nal#%d %d\n", i, nal[i].i_type);
		if (nal[i].i_type != NAL_SEI) {
			memcpy(pps, nal[i].p_payload, nal[i].i_payload);
			pps += nal[i].i_payload;
		} else {
			m->seilen = nal[i].i_payload;
			m->sei = malloc(m->seilen);
			memcpy(m->sei, nal[i].p_payload, m->seilen);
		}
	}
	m->ppslen = pps - m->pps;
}

static void h264enc_encode(h264enc_t *m) {
	x264_picture_t out;

	m->in.img.i_csp = X264_CSP_I420;
	m->in.i_type = X264_TYPE_AUTO;

	x264_encoder_encode(m->x, &m->nal, &m->nnal, &m->in, &out);
	m->nallen = 0;
	int i;
	if (m->seilen)
		m->nallen += m->seilen;
	for (i = 0; i < m->nnal; i++) {
		m->nallen += m->nal[i].i_payload;
	}
}

static void h264enc_copy(h264enc_t *m, uint8_t *p) {
	int i;
	if (m->seilen) {
		memcpy(p, m->sei, m->seilen);
		p += m->seilen;
		m->seilen = 0;
	}
	for (i = 0; i < m->nnal; i++) {
		memcpy(p, m->nal[i].p_payload, m->nal[i].i_payload);
		p += m->nal[i].i_payload;
	}
}

*/
	"C"
	"unsafe"
	"image"
)

type H264Encoder struct {
	m C.h264enc_t
	PPS []byte
}

func NewH264Encoder(w, h int) (m *H264Encoder) {
	m = &H264Encoder{}
	m.m.w = (C.int)(w)
	m.m.h = (C.int)(h)
	C.h264enc_new(&m.m)
	m.PPS = fromCPtr(unsafe.Pointer(m.m.pps), (int)(m.m.ppslen))
	return
}

func (m *H264Encoder) Encode(f *image.YCbCr) (nal []byte) {
	C.x264_picture_init(&m.m.in);
	m.m.in.img.plane[0] = (*C.uint8_t)(unsafe.Pointer(&f.Y[0]));
	m.m.in.img.plane[1] = (*C.uint8_t)(unsafe.Pointer(&f.Cb[0]));
	m.m.in.img.plane[2] = (*C.uint8_t)(unsafe.Pointer(&f.Cr[0]));
	m.m.in.img.i_stride[0] = (C.int)(f.YStride);
	m.m.in.img.i_stride[1] = (C.int)(f.CStride);
	m.m.in.img.i_stride[2] = (C.int)(f.CStride);
	C.h264enc_encode(&m.m)
	nal = make([]byte, m.m.nallen)
	C.h264enc_copy(&m.m, (*C.uint8_t)(unsafe.Pointer(&nal[0])))
	return
}

