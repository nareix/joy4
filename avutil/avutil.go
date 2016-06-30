package avutil

import (
	"io"
	"fmt"
	"bytes"
	"github.com/nareix/av"
	"net/url"
	"os"
	"path"
)

type handlerDemuxer struct {
	av.Demuxer
	r io.ReadCloser
}

func (self handlerDemuxer) Close() error {
	return self.r.Close()
}

type handlerMuxer struct {
	av.Muxer
	w io.WriteCloser
}

func (self handlerMuxer) Close() error {
	return self.w.Close()
}

type RegisterHandler struct {
	Ext string
	ReaderDemuxer func(io.Reader)av.Demuxer
	WriterMuxer func(io.Writer)av.Muxer
	UrlDemuxer func(string)(bool,av.DemuxerCloser,error)
	UrlReader func(string)(bool,io.ReadCloser,error)
	Probe func([]byte)bool
}

type Handlers struct {
	handlers []RegisterHandler
}

func (self *Handlers) Add(fn func(*RegisterHandler)) {
	handler := &RegisterHandler{}
	fn(handler)
	self.handlers = append(self.handlers, *handler)
}

func (self *Handlers) openUrl(u *url.URL, uri string) (r io.ReadCloser, err error) {
	if u == nil {
		r, err = os.Open(uri)
		return
	}
	for _, handler := range self.handlers {
		if handler.UrlReader != nil {
			var ok bool
			if ok, r, err = handler.UrlReader(uri); ok {
				return
			}
		}
	}
	err = fmt.Errorf("avutil: openUrl %s failed", uri)
	return
}

func (self *Handlers) createUrl(u *url.URL, uri string) (w io.WriteCloser, err error) {
	w, err = os.Create(uri)
	return
}

func (self *Handlers) Open(uri string) (demuxer av.DemuxerCloser, err error) {
	for _, handler := range self.handlers {
		if handler.UrlDemuxer != nil {
			var ok bool
			if ok, demuxer, err = handler.UrlDemuxer(uri); ok {
				return
			}
		}
	}

	var r io.ReadCloser
	var ext string
	var u *url.URL
	var uerr error
	if u, uerr = url.Parse(uri); uerr != nil {
		ext = path.Ext(uri)
	} else {
		ext = path.Ext(u.Path)
	}

	if ext != "" {
		for _, handler := range self.handlers {
			if handler.Ext == ext {
				if handler.ReaderDemuxer != nil {
					if r, err = self.openUrl(u, uri); err != nil {
						return
					}
					demuxer = handlerDemuxer{
						Demuxer: handler.ReaderDemuxer(r),
						r: r,
					}
					return
				}
			}
		}
	}

	var probebuf [1024]byte
	if r, err = self.openUrl(u, uri); err != nil {
		return
	}
	if _, err = io.ReadFull(r, probebuf[:]); err != nil {
		return
	}

	for _, handler := range self.handlers {
		if handler.Probe != nil && handler.Probe(probebuf[:]) && handler.ReaderDemuxer != nil {
			demuxer = handlerDemuxer{
				Demuxer: handler.ReaderDemuxer(io.MultiReader(bytes.NewReader(probebuf[:]), r)),
				r: r,
			}
			return
		}
	}

	err = fmt.Errorf("avutil: open %s failed", uri)
	return
}

func (self *Handlers) Create(uri string) (muxer av.MuxerCloser, err error) {
	var ext string
	var u *url.URL
	var uerr error
	if u, uerr = url.Parse(uri); uerr != nil {
		ext = path.Ext(uri)
	} else {
		ext = path.Ext(u.Path)
	}

	if ext != "" {
		for _, handler := range self.handlers {
			if handler.Ext == ext && handler.WriterMuxer != nil {
				var w io.WriteCloser
				if w, err = self.createUrl(u, uri); err != nil {
					return
				}
				muxer = handlerMuxer{
					Muxer: handler.WriterMuxer(w),
					w: w,
				}
				return
			}
		}
	}

	err = fmt.Errorf("avutil: create %s failed", uri)
	return
}

var defaultHandlers = &Handlers{}

func AddHandler(fn func(*RegisterHandler)) {
	defaultHandlers.Add(fn)
}

func Open(url string) (demuxer av.Demuxer, err error) {
	return defaultHandlers.Open(url)
}

func Create(url string) (muxer av.Muxer, err error) {
	return defaultHandlers.Create(url)
}

