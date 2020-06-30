package format

import (
	"github.com/sprucehealth/joy4/av/avutil"
	"github.com/sprucehealth/joy4/format/aac"
	"github.com/sprucehealth/joy4/format/flv"
	"github.com/sprucehealth/joy4/format/mp4"
	"github.com/sprucehealth/joy4/format/rtmp"
	"github.com/sprucehealth/joy4/format/rtsp"
	"github.com/sprucehealth/joy4/format/ts"
)

func RegisterAll() {
	avutil.DefaultHandlers.Add(mp4.Handler)
	avutil.DefaultHandlers.Add(ts.Handler)
	avutil.DefaultHandlers.Add(rtmp.Handler)
	avutil.DefaultHandlers.Add(rtsp.Handler)
	avutil.DefaultHandlers.Add(flv.Handler)
	avutil.DefaultHandlers.Add(aac.Handler)
}
