package ts

import (
	"io"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
)

func Handler(h *avutil.RegisterHandler) {
	h.Ext = ".ts"
	h.ReaderDemuxer = func(r io.Reader) av.Demuxer {
		return &Demuxer{R: r}
	}
	h.WriterMuxer = func(w io.Writer) av.Muxer {
		return NewMuxer(w)
	}
}

