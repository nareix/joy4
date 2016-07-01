package isom

import (
	"bytes"
	"fmt"
	"github.com/nareix/bits"
	"io"
	"io/ioutil"
)

// copied from libavformat/isom.h

const (
	MP4ESDescrTag          = 3
	MP4DecConfigDescrTag   = 4
	MP4DecSpecificDescrTag = 5
)

var debugReader = false
var debugWriter = false

func readDesc(r io.Reader) (tag uint, data []byte, err error) {
	if tag, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	var length uint
	for i := 0; i < 4; i++ {
		var c uint
		if c, err = bits.ReadUIntBE(r, 8); err != nil {
			return
		}
		length = (length << 7) | (c & 0x7f)
		if c&0x80 == 0 {
			break
		}
	}
	data = make([]byte, length)
	if _, err = r.Read(data); err != nil {
		return
	}
	return
}

func writeDesc(w io.Writer, tag uint, data []byte) (err error) {
	if err = bits.WriteUIntBE(w, tag, 8); err != nil {
		return
	}
	length := uint(len(data))
	for i := 3; i > 0; i-- {
		if err = bits.WriteUIntBE(w, (length>>uint(7*i))&0x7f|0x80, 8); err != nil {
			return
		}
	}
	if err = bits.WriteUIntBE(w, length&0x7f, 8); err != nil {
		return
	}
	if _, err = w.Write(data); err != nil {
		return
	}
	return
}

func readESDesc(r io.Reader) (err error) {
	var ES_ID uint
	// ES_ID
	if ES_ID, err = bits.ReadUIntBE(r, 16); err != nil {
		return
	}
	var flags uint
	if flags, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	//streamDependenceFlag
	if flags&0x80 != 0 {
		if _, err = bits.ReadUIntBE(r, 16); err != nil {
			return
		}
	}
	//URL_Flag
	if flags&0x40 != 0 {
		var length uint
		if length, err = bits.ReadUIntBE(r, 8); err != nil {
			return
		}
		if _, err = io.CopyN(ioutil.Discard, r, int64(length)); err != nil {
			return
		}
	}
	//OCRstreamFlag
	if flags&0x20 != 0 {
		if _, err = bits.ReadUIntBE(r, 16); err != nil {
			return
		}
	}
	if debugReader {
		println("readESDesc:", ES_ID, flags)
	}
	return
}

func writeESDesc(w io.Writer, ES_ID uint) (err error) {
	// ES_ID
	if err = bits.WriteUIntBE(w, ES_ID, 16); err != nil {
		return
	}
	// flags
	if err = bits.WriteUIntBE(w, 0, 8); err != nil {
		return
	}
	return
}

func readDescByTag(r io.Reader, targetTag uint) (data []byte, err error) {
	var found bool
	for {
		if tag, _data, err := readDesc(r); err != nil {
			break
		} else {
			if tag == targetTag {
				data = _data
				found = true
			}
			if debugReader {
				println("readDescByTag:", tag, len(_data))
			}
		}
	}
	if !found {
		err = fmt.Errorf("tag not found")
		return
	}
	return
}

// copied from libavformat/isom.c ff_mp4_read_dec_config_descr()
func readDecConfDesc(r io.Reader) (decConfig []byte, err error) {
	var objectId uint
	var streamType uint
	var bufSize uint
	var maxBitrate uint
	var avgBitrate uint

	// objectId
	if objectId, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	// streamType
	if streamType, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	// buffer size db
	if bufSize, err = bits.ReadUIntBE(r, 24); err != nil {
		return
	}
	// max bitrate
	if maxBitrate, err = bits.ReadUIntBE(r, 32); err != nil {
		return
	}
	// avg bitrate
	if avgBitrate, err = bits.ReadUIntBE(r, 32); err != nil {
		return
	}

	if debugReader {
		println("readDecConfDesc:", objectId, streamType, bufSize, maxBitrate, avgBitrate)
	}

	if decConfig, err = readDescByTag(r, MP4DecSpecificDescrTag); err != nil {
		return
	}
	return
}

// copied from libavformat/movenc.c mov_write_esds_tag()
func writeDecConfDesc(w io.Writer, objectId uint, streamType uint, decConfig []byte) (err error) {
	// objectId
	if err = bits.WriteUIntBE(w, objectId, 8); err != nil {
		return
	}
	// streamType
	if err = bits.WriteUIntBE(w, streamType, 8); err != nil {
		return
	}
	// buffer size db
	if err = bits.WriteUIntBE(w, 0, 24); err != nil {
		return
	}
	// max bitrate
	if err = bits.WriteUIntBE(w, 200000, 32); err != nil {
		return
	}
	// avg bitrate
	if err = bits.WriteUIntBE(w, 0, 32); err != nil {
		return
	}
	if err = writeDesc(w, MP4DecSpecificDescrTag, decConfig); err != nil {
		return
	}
	return
}

// copied from libavformat/mov.c ff_mov_read_esds()
func ReadElemStreamDesc(r io.Reader) (decConfig []byte, err error) {
	if debugReader {
		println("ReadElemStreamDesc: start")
	}

	var data []byte
	if data, err = readDescByTag(r, MP4ESDescrTag); err != nil {
		return
	}
	r = bytes.NewReader(data)

	if err = readESDesc(r); err != nil {
		return
	}

	if data, err = readDescByTag(r, MP4DecConfigDescrTag); err != nil {
		return
	}
	r = bytes.NewReader(data)

	if decConfig, err = readDecConfDesc(r); err != nil {
		return
	}

	if debugReader {
		println("ReadElemStreamDesc: end")
	}
	return
}

func WriteElemStreamDesc(w io.Writer, decConfig []byte, trackId uint) (err error) {
	// MP4ESDescrTag(ESDesc MP4DecConfigDescrTag(objectId streamType bufSize avgBitrate MP4DecSpecificDescrTag(decConfig)))

	data := decConfig

	buf := &bytes.Buffer{}
	// 0x40 = ObjectType AAC
	// 0x15 = Audiostream
	writeDecConfDesc(buf, 0x40, 0x15, data)
	data = buf.Bytes()

	buf = &bytes.Buffer{}
	writeDesc(buf, MP4DecConfigDescrTag, data) // 4
	data = buf.Bytes()

	buf = &bytes.Buffer{}
	writeESDesc(buf, trackId)
	buf.Write(data)
	writeDesc(buf, 0x06, []byte{0x02})
	data = buf.Bytes()

	buf = &bytes.Buffer{}
	writeDesc(buf, MP4ESDescrTag, data) // 3
	data = buf.Bytes()

	if _, err = w.Write(data); err != nil {
		return
	}
	return
}
