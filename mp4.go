
package mp4

import (
	"./atom"
	"os"
	"io"
	"log"
	"encoding/hex"
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

func changeMoov(moov *atom.Movie) {
	header := moov.Header

	log.Println("moov: ", header.CreateTime, header.TimeScale, header.Duration)
	log.Println("moov: ", header.PreferredRate, header.PreferredVolume)
	log.Println("moov: ", header.PreviewTime, header.PreviewDuration)
	log.Println("moov: ", header.PosterTime)
	log.Println("moov: ", header.SelectionTime, header.SelectionDuration)
	log.Println("moov: ", header.CurrentTime)
	log.Println("moov: ", header.NextTrackId)
	log.Println("moov: ", header.Matrix)
	header.NextTrackId = 0

	for i, track := range moov.Tracks {
		log.Println("track", i, ":", track.Header.TrackId)
		log.Println("track", i, ":", track.Header.Duration)
		log.Println("track", i, ":", track.Header.Layer, track.Header.AlternateGroup)
		log.Println("track", i, ":", track.Header.Volume)
		log.Println("track", i, ":", track.Header.TrackWidth, track.Header.TrackHeight)
		log.Println("track", i, ":", track.Header.Matrix)

		media := track.Media

		if minf := media.Info; minf != nil {
			if sample := minf.Sample; sample != nil {
				if desc := sample.SampleDesc; desc != nil {
					if avc1Desc := desc.Avc1Desc; avc1Desc != nil {
						if conf := avc1Desc.Conf; conf != nil {
							log.Println("avc1", hex.Dump(conf.Data))
						}
					}

					/*
					if mp4a := desc.Mp4aDesc; mp4a != nil {
						if conf := mp4a.Conf; conf != nil {
							log.Println("mp4a", hex.Dump(conf.Data))
						}
					}
					*/

				}
			}
		}
	}
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

	var outfile *os.File
	if outfile, err = os.Create(filename+".out.mp4"); err != nil {
		return
	}

	for lr.N > 0 {
		var ar *io.LimitedReader

		var cc4 string
		if ar, cc4, err = atom.ReadAtomHeader(lr, ""); err != nil {
			return
		}

		if cc4 == "moov" {
			curPos, _ := outfile.Seek(0, 1)
			origSize := ar.N+8
			var moov *atom.Movie
			if moov, err = atom.ReadMovie(ar); err != nil {
				return
			}
			changeMoov(moov)
			if err = atom.WriteMovie(outfile, moov); err != nil {
				return
			}
			curPosAfterRead, _ := outfile.Seek(0, 1)
			bytesWritten := curPosAfterRead - curPos

			log.Println("regen moov", "tracks nr", len(moov.Tracks),
				"origSize", origSize, "bytesWritten", bytesWritten,
			)

			padSize := origSize - bytesWritten - 8
			aw, _ := atom.WriteAtomHeader(outfile, "free")
			atom.WriteDummy(outfile, int(padSize))
			aw.Close()
		} else {
			var aw *atom.Writer
			if aw, err = atom.WriteAtomHeader(outfile, cc4); err != nil {
				return
			}
			log.Println("copy", cc4)
			if _, err = io.CopyN(aw, ar, ar.N); err != nil {
				return
			}
			if err = aw.Close(); err != nil {
				return
			}
		}

		//log.Println("atom", cc4, "left", lr.N)
		//atom.ReadDummy(ar, int(ar.N))
	}

	if err = outfile.Close(); err != nil {
		return
	}

	return
}

func Create(filename string) (file *File, err error) {
	return
}

