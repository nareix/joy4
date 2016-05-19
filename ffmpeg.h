
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <string.h>

typedef struct {
	AVCodec *codec;
	AVCodecContext *codecCtx;
	AVFrame *frame;
} FFCtx;

int FFCtxFindEncoderByName(FFCtx *ff, const char *name);
int FFCtxFindDecoderByName(FFCtx *ff, const char *name);

