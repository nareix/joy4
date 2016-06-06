package ffmpeg

/*
#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lavresample
#include "ffmpeg.h"
void ffinit() {
	av_register_all();
}
*/
import "C"

const (
	QUIET = int(C.AV_LOG_QUIET)
	PANIC = int(C.AV_LOG_PANIC)
	FATAL = int(C.AV_LOG_FATAL)
	ERROR = int(C.AV_LOG_ERROR)
	WARNING = int(C.AV_LOG_WARNING)
	INFO = int(C.AV_LOG_INFO)
	VERBOSE = int(C.AV_LOG_VERBOSE)
	DEBUG = int(C.AV_LOG_DEBUG)
	TRACE = int(C.AV_LOG_TRACE)
)

func SetLogLevel(level int) {
	C.av_log_set_level(C.int(level))
}

func init() {
	C.ffinit()
}

