package ffmpeg

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lavresample
#include "ffmpeg.h"
void ffinit() {
	av_register_all();
	av_log_set_level(AV_LOG_DEBUG);
}
*/
import "C"

func init() {
	C.ffinit()
}

