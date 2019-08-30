#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavresample/avresample.h>
#include <libavutil/opt.h>
#include <string.h>
#include <libswscale/swscale.h>
#include "ffmpeg.h"

int decode(AVCodecContext *avctx, AVFrame *frame, int *got_frame, AVPacket *pkt)
{
    int ret;

    *got_frame = 0;

    if (pkt) {
        ret = avcodec_send_packet(avctx, pkt);
        // In particular, we don't expect AVERROR(EAGAIN), because we read all
        // decoded frames with avcodec_receive_frame() until done.
        if (ret < 0)
            return ret == AVERROR_EOF ? 0 : ret;
    }

    ret = avcodec_receive_frame(avctx, frame);
    if (ret < 0 && ret != AVERROR(EAGAIN) && ret != AVERROR_EOF)
        return ret;
    if (ret >= 0)
        *got_frame = 1;

    return 0;
}

int encode(AVCodecContext *avctx, AVPacket *pkt, int *got_packet, AVFrame *frame)
{
    int ret;

    *got_packet = 0;

    ret = avcodec_send_frame(avctx, frame);
    if (ret < 0)
        return ret;

    ret = avcodec_receive_packet(avctx, pkt);
    if (!ret)
        *got_packet = 1;
    if (ret == AVERROR(EAGAIN))
        return 0;

    return ret;
}


int wrap_avcodec_decode(AVCodecContext *avctx, AVFrame *frame,uint8_t *data, int size, int *got_frame)
{
	struct AVPacket pkt = {.data = data, .size = size};
    return decode(avctx, frame, got_frame, &pkt);
}

int wrap_avcodec_encode_jpeg(AVCodecContext *pCodecCtx, AVFrame *pFrame,AVPacket *packet) {
    AVCodec *jpegCodec = avcodec_find_encoder(AV_CODEC_ID_MJPEG);
        
    if (!jpegCodec) {
        return -1;
    }
    
    AVCodecContext *jpegContext = avcodec_alloc_context3(jpegCodec);
    if (!jpegContext) {
        return -1;
    }
    
    jpegContext->pix_fmt = pCodecCtx->pix_fmt;
    jpegContext->height = pFrame->height;
    jpegContext->width = pFrame->width;
    jpegContext->time_base= (AVRational){1,25};

    int ret = avcodec_open2(jpegContext, jpegCodec, NULL);
    
    if (ret < 0) {
        avcodec_close(jpegContext);
        return -1;
    }
    
    int gotFrame;
    
    if (encode(jpegContext, packet, &gotFrame, pFrame) < 0) {
        avcodec_close(jpegContext);
        return -1;
    }
    avcodec_close(jpegContext);
    return 0;
}

int wrap_avresample_convert(AVAudioResampleContext *avr, int *out, int outsize, int outcount, int *in, int insize, int incount) {
	return avresample_convert(avr, (void *)out, outsize, outcount, (void *)in, insize, incount);
}