package mp4

import (
	"io"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
)

func Handler(h *avutil.RegisterHandler) {
	h.Ext = ".mp4"
	h.ReaderDemuxer = func(r io.Reader) av.Demuxer {
		return NewDemuxer(r.(io.ReadSeeker))
	}
	h.WriterMuxer = func(w io.Writer) av.Muxer {
		return NewMuxer(w.(io.WriteSeeker))
	}
}

