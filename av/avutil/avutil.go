package avutil

import (
	"io"
	"fmt"
	"bytes"
	"github.com/nareix/joy4/av"
	"net/url"
	"os"
	"path"
)

type handlerDemuxer struct {
	av.Demuxer
	r io.ReadCloser
}

func (self *handlerDemuxer) Close() error {
	return self.r.Close()
}

type handlerMuxer struct {
	av.Muxer
	w io.WriteCloser
	trailerwritten bool
}

func (self *handlerMuxer) WriteTrailer() (err error) {
	if self.trailerwritten {
		return
	}
	if err = self.Muxer.WriteTrailer(); err != nil {
		return
	}
	self.trailerwritten = true
	return
}

func (self *handlerMuxer) Close() (err error) {
	if err = self.WriteTrailer(); err != nil {
		return
	}
	return self.w.Close()
}

type RegisterHandler struct {
	Ext string
	ReaderDemuxer func(io.Reader)av.Demuxer
	WriterMuxer func(io.Writer)av.Muxer
	UrlDemuxer func(string)(bool,av.DemuxCloser,error)
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
	if u != nil && u.Scheme != "" {
		for _, handler := range self.handlers {
			if handler.UrlReader != nil {
				var ok bool
				if ok, r, err = handler.UrlReader(uri); ok {
					return
				}
			}
		}
		err = fmt.Errorf("avutil: openUrl %s failed", uri)
	} else {
		r, err = os.Open(uri)
	}
	return
}

func (self *Handlers) createUrl(u *url.URL, uri string) (w io.WriteCloser, err error) {
	w, err = os.Create(uri)
	return
}

func (self *Handlers) Open(uri string) (demuxer av.DemuxCloser, err error) {
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
	if u, _ = url.Parse(uri); u != nil && u.Scheme != "" {
		ext = path.Ext(u.Path)
	} else {
		ext = path.Ext(uri)
	}

	if ext != "" {
		for _, handler := range self.handlers {
			if handler.Ext == ext {
				if handler.ReaderDemuxer != nil {
					if r, err = self.openUrl(u, uri); err != nil {
						return
					}
					demuxer = &handlerDemuxer{
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
			demuxer = &handlerDemuxer{
				Demuxer: handler.ReaderDemuxer(io.MultiReader(bytes.NewReader(probebuf[:]), r)),
				r: r,
			}
			return
		}
	}

	r.Close()
	err = fmt.Errorf("avutil: open %s failed", uri)
	return
}

func (self *Handlers) Create(uri string) (muxer av.MuxCloser, err error) {
	var ext string
	var u *url.URL
	if u, _ = url.Parse(uri); u != nil && u.Scheme != "" {
		ext = path.Ext(u.Path)
	} else {
		ext = path.Ext(uri)
	}

	if ext != "" {
		for _, handler := range self.handlers {
			if handler.Ext == ext && handler.WriterMuxer != nil {
				var w io.WriteCloser
				if w, err = self.createUrl(u, uri); err != nil {
					return
				}
				muxer = &handlerMuxer{
					Muxer: handler.WriterMuxer(w),
					w: w,
				}
				return
			}
		}
	}

	err = fmt.Errorf("avutil: create muxer %s failed", uri)
	return
}

var DefaultHandlers = &Handlers{}

func AddHandler(fn func(*RegisterHandler)) {
	DefaultHandlers.Add(fn)
}

func Open(url string) (demuxer av.DemuxCloser, err error) {
	return DefaultHandlers.Open(url)
}

func Create(url string) (muxer av.MuxCloser, err error) {
	return DefaultHandlers.Create(url)
}

