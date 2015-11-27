
package mp4

import (
	"github.com/nareix/mp4/atom"
	"os"
	"io"
	"fmt"
	"log"
	"encoding/hex"
)

/*
func TestRewrite(filename string) (err error) {
	var infile *os.File
	if infile, err = os.Open(filename); err != nil {
		return
	}

	var outfile *os.File
	if outfile, err = os.Open(filename+".out.mp4"); err != nil {
		return
	}

	return
}
*/

func getAVCDecoderConfRecordByTrack(track *atom.Track) (record *atom.AVCDecoderConfRecord) {
	if media := track.Media; media != nil {
		if info := media.Info; info != nil {
			if sample := info.Sample; sample != nil {
				if desc := sample.SampleDesc; desc != nil {
					if avc1 := desc.Avc1Desc; avc1 != nil {
						if conf := avc1.Conf; conf != nil {
							return &conf.Record
						}
					}
				}
			}
		}
	}
	return
}

func ProbeFile(filename string) (err error) {
	var osfile *os.File
	if osfile, err = os.Open(filename); err != nil {
		return
	}

	var finfo os.FileInfo
	if finfo, err = osfile.Stat(); err != nil {
		return
	}

	dumper := &atom.Dumper{}
	var moov *atom.Movie

	lr := &io.LimitedReader{R: osfile, N: finfo.Size()}
	for lr.N > 0 {
		var ar *io.LimitedReader

		var cc4 string
		if ar, cc4, err = atom.ReadAtomHeader(lr, ""); err != nil {
			log.Println("read atom failed")
			return
		}

		if cc4 == "moov" {
			if moov, err = atom.ReadMovie(ar); err != nil {
				log.Println("read '%s' atom failed", cc4)
				return
			}
			if false {
				atom.WalkMovie(dumper, moov)
			}
		} else {
			fmt.Println("atom:", cc4)
		}

		if _, err = atom.ReadDummy(lr, int(ar.N)); err != nil {
			return
		}
	}

	if moov == nil {
		err = fmt.Errorf("'moov' atom not found")
		log.Println("'moov' atom not found")
		return
	}

	if len(moov.Tracks) > 0 {
		track := moov.Tracks[0]
		record := getAVCDecoderConfRecordByTrack(track)

		if record != nil && len(record.SPS) > 0 {
			sps := record.SPS[0]
			if len(sps) > 1 {
				sps = sps[1:]
				log.Println(hex.Dump(sps))
				var info *atom.H264SPSInfo
				if info, err = atom.ParseH264SPS(sps); err != nil {
					return
				}
				log.Println(info)
			}
		}
	}

	return
}

func TestConvert(filename string) (err error) {
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

			//log.Println(moov.Tracks[0].Media.Info.Data.Refer)

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

			outcc4 := cc4
			if outcc4 != "mdat" {
				log.Println("omit", cc4)
				outcc4 = "free"
			} else {
				log.Println("copy", cc4)
			}

			var aw *atom.Writer
			if aw, err = atom.WriteAtomHeader(outfile, outcc4); err != nil {
				return
			}

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

