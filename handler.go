package mp4

import (
	"io"
	"github.com/nareix/av"
	"github.com/nareix/av/avutil"
)

func Handler(h *avutil.RegisterHandler) {
	h.Ext = ".mp4"
	h.ReaderDemuxer = func(r io.Reader) av.Demuxer {
		return &Demuxer{R: r.(io.ReadSeeker)}
	}
	h.WriterMuxer = func(w io.Writer) av.Muxer {
		return &Muxer{W: w.(io.WriteSeeker)}
	}
}

