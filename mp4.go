
package mp4

import (
	"./atom"
	"os"
	"io"
	"log"
)

type File struct {
}

func (self *File) AddAvcc(avcc *Avcc) {
}

func (self *File) AddMp4a(mp4a *Mp4a) {
}

func (self *File) GetAvcc() (avcc []*Avcc) {
	return
}

func (self *File) GetMp4a() (mp4a []*Mp4a) {
	return
}

func (self *File) Sync() {
}

func (self *File) Close() {
}

func Open(filename string) (file *File, err error) {
	var osfile *os.File
	if osfile, err = os.Open(filename); err != nil {
		return
	}

	var finfo os.FileInfo
	if finfo, err = osfile.Stat(); err != nil {
		return
	}
	log.Println("filesize", finfo.Size())

	lr := &io.LimitedReader{R: osfile, N: finfo.Size()}

	for lr.N > 0 {
		var r *io.LimitedReader
		var cc4 string
		if r, cc4, err = atom.ReadAtomHeader(lr, ""); err != nil {
			return
		}
		if cc4 == "moov" {
			var moov *atom.Movie
			if moov, err = atom.ReadMovie(r); err != nil {
				return
			}
			log.Println("tracks nr", len(moov.Tracks))
		}
		log.Println("atom", cc4, "left", lr.N)
		atom.ReadDummy(r, int(r.N))
	}

	if _, err = os.Create(filename+".out.mp4"); err != nil {
		return
	}

	return
}

func Create(filename string) (file *File, err error) {
	return
}

