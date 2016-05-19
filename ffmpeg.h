
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <string.h>

typedef struct {
	AVCodec *codec;
	AVCodecContext *codecCtx;
	AVFrame *frame;
} FFCtx;

