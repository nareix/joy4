package mp4

import (
	"io"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
)

var CodecTypes = []av.CodecType{av.H264, av.AAC}

func Handler(h *avutil.RegisterHandler) {
	h.Ext = ".mp4"

	h.ReaderDemuxer = func(r io.Reader) av.Demuxer {
		return NewDemuxer(r.(io.ReadSeeker))
	}

	h.WriterMuxer = func(w io.Writer) av.Muxer {
		return NewMuxer(w.(io.WriteSeeker))
	}

	h.CodecTypes = CodecTypes
}

