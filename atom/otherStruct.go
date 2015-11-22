
package atom

import (
	_"io"
	_"bytes"
	_"log"
	_"encoding/hex"
)

/*
type VideoSampleDesc struct {
	VideoSampleDescHeader
	AVCDecoderConf []byte
}

func ReadVideoSampleDesc(r *io.LimitedReader) (res *VideoSampleDesc, err error) {
	self := &VideoSampleDesc{}

	if self.VideoSampleDescHeader, err = ReadVideoSampleDescHeader(r); err != nil {
		return
	}

	for r.N > 0 {
		var cc4 string
		var ar *io.LimitedReader
		if ar, cc4, err = ReadAtomHeader(r, ""); err != nil {
			return
		}

		if false {
			log.Println("VideoSampleDesc:", cc4, ar.N)
			//log.Println("VideoSampleDesc:", "avcC", len(self.AVCDecoderConf))
		}

		switch cc4 {
			case "avcC": {
				if self.AVCDecoderConf, err = ReadBytes(ar, int(ar.N)); err != nil {
					return
				}
			}
		}

		if _, err = ReadDummy(ar, int(ar.N)); err != nil {
			return
		}
	}

	res = self
	return
}
*/

