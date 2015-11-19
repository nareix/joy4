
package mp4

import (
	"./atom"
	"os"
)

type File struct {
	moov *atom.Moov
	ftyp *atom.Ftyp
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

	var entry atom.Atom
	file = &File{}
	r := atom.Reader{osfile}

	if entry, err = r.ReadAtom(&atom.Ftyp{}); err != nil {
		return
	}
	file.ftyp = entry.(*atom.Ftyp)

	if entry, err = r.ReadAtom(&atom.Moov{}); err != nil {
		return
	}
	file.moov = entry.(*atom.Moov)

	return
}

func Create(filename string) (file *File, err error) {
	return
}

