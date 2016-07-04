package format

import (
	"github.com/nareix/joy4/format/mp4"
	"github.com/nareix/joy4/format/ts"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/nareix/joy4/format/rtsp"
	"github.com/nareix/joy4/format/flv"
	"github.com/nareix/joy4/format/aac"
	"github.com/nareix/joy4/av/avutil"
)

func RegisterAll() {
	avutil.AddHandler(mp4.Handler)
	avutil.AddHandler(ts.Handler)
	avutil.AddHandler(rtmp.Handler)
	avutil.AddHandler(rtsp.Handler)
	avutil.AddHandler(flv.Handler)
	avutil.AddHandler(aac.Handler)
}

