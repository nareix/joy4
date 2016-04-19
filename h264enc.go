package codec

import (

	/*
		#include <stdio.h>
		#include <stdlib.h>
		#include <stdint.h>
		#include <string.h>
		#include <libavcodec/avcodec.h>
		#include <libavformat/avformat.h>
		#include <libavutil/avutil.h>

		typedef struct {
			int w, h;
			int pixfmt;
			char *preset[2];
			char *profile;
			int bitrate;
			int got;
			AVCodec *c;
			AVCodecContext *ctx;
			AVFrame *f;
			AVPacket pkt;
		} h264enc_t;

		static int h264enc_new(h264enc_t *m) {
			m->c = avcodec_find_encoder(CODEC_ID_H264);
			m->ctx = avcodec_alloc_context3(m->c);
			m->ctx->width = m->w;
			m->ctx->height = m->h;
			m->ctx->bit_rate = m->bitrate;
			m->ctx->pix_fmt = m->pixfmt;
			m->ctx->flags |= CODEC_FLAG_GLOBAL_HEADER;
			m->f = av_frame_alloc();
			return avcodec_open2(m->ctx, m->c, NULL);
		}

	*/
	"C"
	"errors"
	"image"
	"strings"
	"unsafe"
	//"log"
)

type H264Encoder struct {
	m      C.h264enc_t
	Header []byte
	Pixfmt image.YCbCrSubsampleRatio
	W, H   int
}

func NewH264Encoder(
	w, h int,
	pixfmt image.YCbCrSubsampleRatio,
	opts ...string,
) (m *H264Encoder, err error) {
	m = &H264Encoder{}
	m.m.w = (C.int)(w)
	m.m.h = (C.int)(h)
	m.W = w
	m.H = h
	m.Pixfmt = pixfmt
	switch pixfmt {
	case image.YCbCrSubsampleRatio444:
		m.m.pixfmt = C.PIX_FMT_YUV444P
	case image.YCbCrSubsampleRatio422:
		m.m.pixfmt = C.PIX_FMT_YUV422P
	case image.YCbCrSubsampleRatio420:
		m.m.pixfmt = C.PIX_FMT_YUV420P
	}
	for _, opt := range opts {
		a := strings.Split(opt, ",")
		switch {
		case a[0] == "preset" && len(a) == 3:
			m.m.preset[0] = C.CString(a[1])
			m.m.preset[1] = C.CString(a[2])
		case a[0] == "profile" && len(a) == 2:
			m.m.profile = C.CString(a[1])
		}
	}
	r := C.h264enc_new(&m.m)
	if int(r) < 0 {
		err = errors.New("open encoder failed")
		return
	}
	m.Header = fromCPtr(unsafe.Pointer(m.m.ctx.extradata), (int)(m.m.ctx.extradata_size))
	//m.Header = fromCPtr(unsafe.Pointer(m.m.pps), (int)(m.m.ppslen))
	return
}

type h264Out struct {
	Data []byte
	Key  bool
}

func (m *H264Encoder) Encode(img *image.YCbCr) (out h264Out, err error) {
	var f *C.AVFrame
	if img == nil {
		f = nil
	} else {
		if img.SubsampleRatio != m.Pixfmt {
			err = errors.New("image pixfmt not match")
			return
		}
		if img.Rect.Dx() != m.W || img.Rect.Dy() != m.H {
			err = errors.New("image size not match")
			return
		}
		f = m.m.f
		f.data[0] = (*C.uint8_t)(unsafe.Pointer(&img.Y[0]))
		f.data[1] = (*C.uint8_t)(unsafe.Pointer(&img.Cb[0]))
		f.data[2] = (*C.uint8_t)(unsafe.Pointer(&img.Cr[0]))
		f.linesize[0] = (C.int)(img.YStride)
		f.linesize[1] = (C.int)(img.CStride)
		f.linesize[2] = (C.int)(img.CStride)
	}

	C.av_init_packet(&m.m.pkt)
	r := C.avcodec_encode_video2(m.m.ctx, &m.m.pkt, f, &m.m.got)
	defer C.av_free_packet(&m.m.pkt)
	if int(r) < 0 {
		err = errors.New("encode failed")
		return
	}
	if m.m.got == 0 {
		err = errors.New("no picture")
		return
	}
	if m.m.pkt.size == 0 {
		err = errors.New("packet size == 0")
		return
	}

	out.Data = make([]byte, m.m.pkt.size)
	C.memcpy(
		unsafe.Pointer(&out.Data[0]),
		unsafe.Pointer(m.m.pkt.data),
		(C.size_t)(m.m.pkt.size),
	)
	out.Key = (m.m.pkt.flags & C.AV_PKT_FLAG_KEY) != 0

	return
}
