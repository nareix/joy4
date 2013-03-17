
package rtmp

import (
	"io"
	"encoding/binary"
)

var (
	AMF_NUMBER             = 0x00
	AMF_BOOLEAN            = 0x01
	AMF_STRING             = 0x02
	AMF_OBJECT             = 0x03
	AMF_NULL               = 0x05
	AMF_ARRAY_NULL         = 0x06
	AMF_MIXED_ARRAY        = 0x08
	AMF_END                = 0x09
	AMF_ARRAY              = 0x0a

	AMF_INT8               = 0x0100
	AMF_INT16              = 0x0101
	AMF_INT32              = 0x0102
	AMF_VARIANT_           = 0x0103
)

type AMFObj struct {
	atype int
	str string
	i int
	buf []byte
	obj map[string]AMFObj
	f64 float64
}

func ReadAMF(r io.Reader) (a AMFObj) {
	a.atype = ReadInt(r, 1)
	switch (a.atype) {
	case AMF_STRING:
		n := ReadInt(r, 2)
		b := ReadBuf(r, n)
		a.str = string(b)
	case AMF_NUMBER:
		binary.Read(r, binary.BigEndian, &a.f64)
	case AMF_BOOLEAN:
		a.i = ReadInt(r, 1)
	case AMF_MIXED_ARRAY:
		ReadInt(r, 4)
		fallthrough
	case AMF_OBJECT:
		a.obj = map[string]AMFObj{}
		for {
			n := ReadInt(r, 2)
			if n == 0 {
				break
			}
			name := string(ReadBuf(r, n))
			a.obj[name] = ReadAMF(r)
		}
	case AMF_ARRAY, AMF_VARIANT_:
		panic("amf: read: unsupported array or variant")
	case AMF_INT8:
		a.i = ReadInt(r, 1)
	case AMF_INT16:
		a.i = ReadInt(r, 2)
	case AMF_INT32:
		a.i = ReadInt(r, 4)
	}
	return
}

func WriteAMF(r io.Writer, a AMFObj) {
	WriteInt(r, a.atype, 1)
	switch (a.atype) {
	case AMF_STRING:
		WriteInt(r, len(a.str), 2)
		r.Write([]byte(a.str))
	case AMF_NUMBER:
		binary.Write(r, binary.BigEndian, a.f64)
	case AMF_BOOLEAN:
		WriteInt(r, a.i, 1)
	case AMF_MIXED_ARRAY:
		r.Write(a.buf[:4])
	case AMF_OBJECT:
		for name, val := range a.obj {
			WriteInt(r, len(name), 2)
			r.Write([]byte(name))
			WriteAMF(r, val)
		}
		WriteInt(r, 9, 3)
	case AMF_ARRAY, AMF_VARIANT_:
		panic("amf: write unsupported array, var")
	case AMF_INT8:
		WriteInt(r, a.i, 1)
	case AMF_INT16:
		WriteInt(r, a.i, 2)
	case AMF_INT32:
		WriteInt(r, a.i, 4)
	}
}

