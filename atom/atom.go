
package atom

import (
	"log"
)

type Atom interface {
	CC4() string
	Read(Reader) error
}

type Ftyp struct {
}

func (self Ftyp) CC4() string {
	return "ftyp"
}

func (self Ftyp) Read(r Reader) (err error) {
	log.Println("read ftyp")
	return
}

type Moov struct {
	*Mvhd
	Trak []*Trak
}

func (self Moov) CC4() string {
	return "moov"
}

func (self *Moov) Read(r Reader) (err error) {
	var atom Atom
	if atom, err = r.ReadAtom(&Mvhd{}); err != nil {
		return
	}
	self.Mvhd = atom.(*Mvhd)

	for {
		if atom, err := r.ReadAtom(&Trak{}); err != nil {
			break
		} else {
			self.Trak = append(self.Trak, atom.(*Trak))
		}
	}
	return
}

type Mvhd struct {
}

func (self Mvhd) CC4() string {
	return "mvhd"
}

func (self Mvhd) Read(r Reader) (err error) {
	return
}

type Tkhd struct {
}

func (self Tkhd) CC4() string {
	return "tkhd"
}

func (self *Tkhd) Read(r Reader) (err error) {
	return
}

type Minf struct {
}

type Mdia struct {
	*Minf
}

type Trak struct {
	Tkhd []*Tkhd
	*Mdia
}

func (self Trak) CC4() string {
	return "tkhd"
}

func (self *Trak) Read(r Reader) (err error) {
	return
}

// Time-to-Sample Atoms
type Stts struct {
}

// Composition Offset Atom
type Ctts struct {
}

// Sync Sample Atoms (Keyframe)
type Stss struct {
}

// Sample-to-Chunk Atoms
type Stsc struct {
}

// Sample Size Atoms
type Stsz struct {
}

// Chunk Offset Atoms
type Stco struct {
}

