#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavresample/avresample.h>
#include <libavutil/opt.h>
#include <string.h>
#include <libswscale/swscale.h>
#include "ffmpeg.h"

int wrap_avcodec_decode_video2(AVCodecContext *avctx, AVFrame *frame,uint8_t *data, int size, int *got_frame)
{
    int ret;
	struct AVPacket pkt = {.data = data, .size = size};

    *got_frame = 0;

    if (data) {
        ret = avcodec_send_packet(avctx, &pkt);
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

    if (avcodec_encode_video2(jpegContext, packet, pFrame, &gotFrame) < 0) {
        avcodec_close(jpegContext);
        return -1;
    }
        
    avcodec_close(jpegContext);
    return 0;
}
