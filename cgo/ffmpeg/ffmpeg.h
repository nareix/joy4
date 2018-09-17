
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavresample/avresample.h>
#include <libavutil/opt.h>
#include <string.h>
#include <libswscale/swscale.h>
#include <libavutil/imgutils.h>

typedef struct {
	AVCodec *codec;
	AVCodecContext *codecCtx;
	AVFrame *frame;
	AVDictionary *options;
	int profile;
} FFCtx;

static inline int avcodec_profile_name_to_int(AVCodec *codec, const char *name) {
	const AVProfile *p;
	for (p = codec->profiles; p != NULL && p->profile != FF_PROFILE_UNKNOWN; p++)
		if (!strcasecmp(p->name, name))
			return p->profile;
	return FF_PROFILE_UNKNOWN;
}

