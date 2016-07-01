package ts

import (
	"io"
	"github.com/nareix/av"
	"github.com/nareix/av/avutil"
)

func Handler(h *avutil.RegisterHandler) {
	h.Ext = ".ts"
	h.ReaderDemuxer = func(r io.Reader) av.Demuxer {
		return &Demuxer{R: r}
	}
	h.WriterMuxer = func(w io.Writer) av.Muxer {
		return &Muxer{W: w}
	}
}

