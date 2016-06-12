package atom

import (
	"io"
)

func WalkFile(w Walker, r io.Reader) (err error) {
	var moov *Movie
	var moof *MovieFrag

	for {
		var lr *io.LimitedReader
		var cc4 string
		if lr, cc4, err = ReadAtomHeader(lr, ""); err != nil {
			return
		}

		switch cc4 {
		case "moov":
			if moov, err = ReadMovie(lr); err != nil {
				return
			}
			WalkMovie(w, moov)

		case "moof":
			if moof, err = ReadMovieFrag(lr); err != nil {
				return
			}
			WalkMovieFrag(w, moof)

		case "mdat":
			w.StartStruct("MovieData")
			w.Name("Length")
			w.Int64(lr.N)
			w.EndStruct()
		}

		if _, err = ReadDummy(r, int(lr.N)); err != nil {
			return
		}
	}

	return
}

