package flvio

import (
	"math"
	"fmt"
	"time"
	"github.com/nareix/pio"
)

type AMFMap map[string]interface{}
type AMFArray []interface{}
type AMFECMAArray map[string]interface{}

func readBEFloat64(r *pio.Reader) (f float64, err error) {
	var u uint64
	if u, err = r.ReadU64BE(); err != nil {
		return
	}
	f = math.Float64frombits(u)
	return
}

func writeBEFloat64(w *pio.Writer, f float64) (err error) {
	u := math.Float64bits(f)
	if err = w.WriteU64BE(u); err != nil {
		return
	}
	return
}

const (
	amf3undefinedmarker = iota
	amf3nullmarker
	amf3falsemarker
	amf3truemarker
	amf3integermarker
	amf3doublemarker
	amf3stringmarker
	amf3xmldocmarker
	amf3datemarker
	amf3arraymarker
	amf3objectmarker
	amf3xmlmarker
	amf3bytearraymarker
	amf3vectorintmarker
	amf3vectoruintmarker
	amf3vectordoublemarker
	amf3vectorobjectmarker
	amf3dictionarymarker
)

const (
	numbermarker = iota
	booleanmarker
	stringmarker
	objectmarker
	movieclipmarker
	nullmarker
	undefinedmarker
	referencemarker
	ecmaarraymarker
	objectendmarker
	strictarraymarker
	datemarker
	longstringmarker
	unsupportedmarker
	recordsetmarker
	xmldocumentmarker
	typedobjectmarker
	avmplusobjectmarker
)

func writeAMF0Number(w *pio.Writer, f float64) (err error) {
	if err = w.WriteU8(numbermarker); err != nil {
		return
	}
	if err = writeBEFloat64(w, f); err != nil {
		return
	}
	return
}

func WriteAMF0Val(w *pio.Writer, _val interface{}) (err error) {
	switch val := _val.(type) {
	case int8:
		return writeAMF0Number(w, float64(val))
	case int16:
		return writeAMF0Number(w, float64(val))
	case int32:
		return writeAMF0Number(w, float64(val))
	case int64:
		return writeAMF0Number(w, float64(val))
	case int:
		return writeAMF0Number(w, float64(val))
	case uint8:
		return writeAMF0Number(w, float64(val))
	case uint16:
		return writeAMF0Number(w, float64(val))
	case uint32:
		return writeAMF0Number(w, float64(val))
	case uint64:
		return writeAMF0Number(w, float64(val))
	case uint:
		return writeAMF0Number(w, float64(val))
	case float32:
		return writeAMF0Number(w, float64(val))
	case float64:
		return writeAMF0Number(w, val)

	case string:
		u := len(val)
		if u <= 65536 {
			if err = w.WriteU8(stringmarker); err != nil {
				return
			}
			if err = w.WriteU16BE(uint16(u)); err != nil {
				return
			}
		} else {
			if err = w.WriteU8(longstringmarker); err != nil {
				return
			}
			if err = w.WriteU32BE(uint32(u)); err != nil {
				return
			}
		}
		if _, err = w.Write([]byte(val)); err != nil {
			return
		}

	case AMFECMAArray:
		if err = w.WriteU8(ecmaarraymarker); err != nil {
			return
		}
		if err = w.WriteU32BE(uint32(len(val))); err != nil {
			return
		}
		for k, v := range val {
			if err = w.WriteU16BE(uint16(len(k))); err != nil {
				return
			}
			if _, err = w.Write([]byte(k)); err != nil {
				return
			}
			if err = WriteAMF0Val(w, v); err != nil {
				return
			}
		}
		if err = w.WriteU24BE(0x000009); err != nil {
			return
		}

	case AMFMap:
		if err = w.WriteU8(objectmarker); err != nil {
			return
		}
		for k, v := range val {
			if len(k) > 0 {
				if err = w.WriteU16BE(uint16(len(k))); err != nil {
					return
				}
				if _, err = w.Write([]byte(k)); err != nil {
					return
				}
				if err = WriteAMF0Val(w, v); err != nil {
					return
				}
			}
		}
		if err = w.WriteU24BE(0x000009); err != nil {
			return
		}

	case AMFArray:
		if err = w.WriteU8(strictarraymarker); err != nil {
			return
		}
		if err = w.WriteU32BE(uint32(len(val))); err != nil {
			return
		}
		for _, v := range val {
			if err = WriteAMF0Val(w, v); err != nil {
				return
			}
		}

	case time.Time:
		if err = w.WriteU8(datemarker); err != nil {
			return
		}
		u := val.UnixNano()
		f := float64(u/1000000)
		if err = writeBEFloat64(w, f); err != nil {
			return
		}
		if err = w.WriteU16BE(0); err != nil {
			return
		}

	case bool:
		if err = w.WriteU8(booleanmarker); err != nil {
			return
		}
		var u uint8
		if val {
			u = 1
		} else {
			u = 0
		}
		if err = w.WriteU8(u); err != nil {
			return
		}

	case nil:
		if err = w.WriteU8(nullmarker); err != nil {
			return
		}

	default:
		err = fmt.Errorf("amf0: write: invalid val=%v", val)
		return
	}

	return
}

func ReadAMF0Val(r *pio.Reader) (val interface{}, err error) {
	var marker uint8
	if marker, err = r.ReadU8(); err != nil {
		return
	}

	switch marker {
	case numbermarker:
		if val, err = readBEFloat64(r); err != nil {
			return
		}

	case booleanmarker:
		var u8 uint8
		if u8, err = r.ReadU8(); err != nil {
			return
		}
		val = u8 != 0

	case stringmarker:
		var length uint16
		if length, err = r.ReadU16BE(); err != nil {
			return
		}
		var b []byte
		if b, err = r.ReadBytes(int(length)); err != nil {
			return
		}
		val = string(b)

	case objectmarker:
		obj := AMFMap{}
		for {
			var length uint16
			if length, err = r.ReadU16BE(); err != nil {
				return
			}
			if length == 0 {
				break
			}
			var b []byte
			if b, err = r.ReadBytes(int(length)); err != nil {
				return
			}
			okey := string(b)
			var oval interface{}
			if oval, err = ReadAMF0Val(r); err != nil {
				return
			}
			obj[okey] = oval
		}
		if _, err = r.ReadU8(); err != nil {
			return
		}
		val = obj

	case nullmarker:
	case undefinedmarker:

	case ecmaarraymarker:
		var count uint32
		if count, err = r.ReadU32BE(); err != nil {
			return
		}
		obj := AMFMap{}
		for ; count > 0; count-- {
			var length uint16
			if length, err = r.ReadU16BE(); err != nil {
				return
			}
			var b []byte
			if b, err = r.ReadBytes(int(length)); err != nil {
				return
			}
			okey := string(b)
			var oval interface{}
			if oval, err = ReadAMF0Val(r); err != nil {
				return
			}
			obj[okey] = oval
		}
		if _, err = r.Discard(3); err != nil {
			return
		}
		val = obj

	case objectendmarker:
		if _, err = r.Discard(3); err != nil {
			return
		}

	case strictarraymarker:
		var count uint32
		if count, err = r.ReadU32BE(); err != nil {
			return
		}
		obj := make(AMFArray, count)
		for i := 0; i < int(count); i++ {
			if obj[i], err = ReadAMF0Val(r); err != nil {
				return
			}
		}
		val = obj

	case datemarker:
		var ts float64
		if ts, err = readBEFloat64(r); err != nil {
			return
		}
		if _, err = r.ReadU16BE(); err != nil {
			return
		}
		val = time.Unix(int64(ts/1000), (int64(ts)%1000)*1000000)

	case longstringmarker:
		var length uint32
		if length, err = r.ReadU32BE(); err != nil {
			return
		}
		var b []byte
		if b, err = r.ReadBytes(int(length)); err != nil {
			return
		}
		val = string(b)

	default:
		err = fmt.Errorf("amf0: read: unspported marker=%d", marker)
		return
	}

	return
}

