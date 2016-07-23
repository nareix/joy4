package flvio

import (
	"strings"
	"math"
	"fmt"
	"time"
	"github.com/nareix/pio"
)

type AMF0ParseError struct {
	Offset int
	Message string
	Next *AMF0ParseError
}

func (self *AMF0ParseError) Error() string {
	s := []string{}
	for p := self; p != nil; p = p.Next {
		s = append(s, fmt.Sprintf("%s:%d", p.Message, p.Offset))
	}
	return strings.Join(s, ",")
}

func amf0ParseErr(message string, offset int, err error) error {
	next, _ := err.(*AMF0ParseError)
	return &AMF0ParseError{
		Offset: offset,
		Message: message,
		Next: next,
	}
}

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

func parseBEFloat64(b []byte) float64 {
	return math.Float64frombits(pio.U64BE(b))
}

func writeBEFloat64(w *pio.Writer, f float64) (err error) {
	u := math.Float64bits(f)
	if err = w.WriteU64BE(u); err != nil {
		return
	}
	return
}

func fillBEFloat64(b []byte, f float64) int {
	pio.PutU64BE(b, math.Float64bits(f))
	return 8
}

func writeAMF0Number(w *pio.Writer, f float64) (err error) {
	if err = w.WriteU8(numbermarker); err != nil {
		return
	}
	if err = writeBEFloat64(w, f); err != nil {
		return
	}
	return
}

const lenAMF0Number = 9

func fillAMF0Number(b []byte, f float64) int {
	b[0] = numbermarker
	fillBEFloat64(b[1:], f)
	return lenAMF0Number
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

func LenAMF0Val(_val interface{}) (n int) {
	switch val := _val.(type) {
	case int8:
		n += lenAMF0Number
	case int16:
		n += lenAMF0Number
	case int32:
		n += lenAMF0Number
	case int64:
		n += lenAMF0Number
	case int:
		n += lenAMF0Number
	case uint8:
		n += lenAMF0Number
	case uint16:
		n += lenAMF0Number
	case uint32:
		n += lenAMF0Number
	case uint64:
		n += lenAMF0Number
	case uint:
		n += lenAMF0Number
	case float32:
		n += lenAMF0Number
	case float64:
		n += lenAMF0Number

	case string:
		u := len(val)
		if u <= 65536 {
			n += 3
		} else {
			n += 5
		}
		n += int(u)

	case AMFECMAArray:
		n += 5
		for k, v := range val {
			n += 2+len(k)
			n += LenAMF0Val(v)
		}
		n += 3

	case AMFMap:
		n++
		for k, v := range val {
			if len(k) > 0 {
				n += 2+len(k)
				n += LenAMF0Val(v)
			}
		}
		n += 3

	case AMFArray:
		n += 5
		for _, v := range val {
			n += LenAMF0Val(v)
		}

	case time.Time:
		n += 1+8+2

	case bool:
		n += 2

	case nil:
		n++
	}

	return
}

func FillAMF0Val(b []byte, _val interface{}) (n int) {
	switch val := _val.(type) {
	case int8:
		n += fillAMF0Number(b[n:], float64(val))
	case int16:
		n += fillAMF0Number(b[n:], float64(val))
	case int32:
		n += fillAMF0Number(b[n:], float64(val))
	case int64:
		n += fillAMF0Number(b[n:], float64(val))
	case int:
		n += fillAMF0Number(b[n:], float64(val))
	case uint8:
		n += fillAMF0Number(b[n:], float64(val))
	case uint16:
		n += fillAMF0Number(b[n:], float64(val))
	case uint32:
		n += fillAMF0Number(b[n:], float64(val))
	case uint64:
		n += fillAMF0Number(b[n:], float64(val))
	case uint:
		n += fillAMF0Number(b[n:], float64(val))
	case float32:
		n += fillAMF0Number(b[n:], float64(val))
	case float64:
		n += fillAMF0Number(b[n:], float64(val))

	case string:
		u := len(val)
		if u <= 65536 {
			b[n] = stringmarker
			n++
			pio.PutU16BE(b[n:], uint16(u))
			n += 2
		} else {
			b[n] = longstringmarker
			n++
			pio.PutU32BE(b[n:], uint32(u))
			n += 4
		}
		copy(b[n:], []byte(val))
		n += len(val)

	case AMFECMAArray:
		b[n] = ecmaarraymarker
		n++
		pio.PutU32BE(b[n:], uint32(len(val)))
		n += 4
		for k, v := range val {
			pio.PutU16BE(b[n:], uint16(len(k)))
			n += 2
			copy(b[n:], []byte(k))
			n += len(k)
			n += FillAMF0Val(b[n:], v)
		}
		pio.PutU24BE(b[n:], 0x000009)
		n += 3

	case AMFMap:
		b[n] = objectmarker
		n++
		for k, v := range val {
			if len(k) > 0 {
				pio.PutU16BE(b[n:], uint16(len(k)))
				n += 2
				copy(b[n:], []byte(k))
				n += len(k)
				n += FillAMF0Val(b[n:], v)
			}
		}
		pio.PutU24BE(b[n:], 0x000009)
		n += 3

	case AMFArray:
		b[n] = strictarraymarker
		n++
		pio.PutU32BE(b[n:], uint32(len(val)))
		n += 4
		for _, v := range val {
			n += FillAMF0Val(b[n:], v)
		}

	case time.Time:
		b[n] = datemarker
		n++
		u := val.UnixNano()
		f := float64(u/1000000)
		n += fillBEFloat64(b[n:], f)
		pio.PutU16BE(b[n:], uint16(0))
		n += 2

	case bool:
		b[n] = booleanmarker
		n++
		var u uint8
		if val {
			u = 1
		} else {
			u = 0
		}
		b[n] = u
		n++

	case nil:
		b[n] = nullmarker
		n++
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

func ParseAMF0Val(b []byte) (val interface{}, n int, err error) {
	return parseAMF0Val(b, 0)
}

func parseAMF0Val(b []byte, offset int) (val interface{}, n int, err error) {
	if len(b) < n+1 {
		err = amf0ParseErr("marker", offset+n, err)
		return
	}
	marker := b[n]
	n++

	switch marker {
	case numbermarker:
		if len(b) < n+8 {
			err = amf0ParseErr("number", offset+n, err)
			return
		}
		val = parseBEFloat64(b[n:])
		n += 8

	case booleanmarker:
		if len(b) < n+1 {
			err = amf0ParseErr("boolean", offset+n, err)
			return
		}
		val = b[n] != 0
		n++

	case stringmarker:
		if len(b) < n+2 {
			err = amf0ParseErr("string.length", offset+n, err)
			return
		}
		length := int(pio.U16BE(b[n:]))
		n += 2

		if len(b) < n+length {
			err = amf0ParseErr("string.body", offset+n, err)
			return
		}
		val = string(b[n:n+length])
		n += length

	case objectmarker:
		obj := AMFMap{}
		for {
			if len(b) < n+2 {
				err = amf0ParseErr("object.key.length", offset+n, err)
				return
			}
			length := int(pio.U16BE(b[n:]))
			n += 2
			if length == 0 {
				break
			}

			if len(b) < n+length {
				err = amf0ParseErr("object.key.body", offset+n, err)
				return
			}
			okey := string(b[n:n+length])
			n += length

			var nval int
			var oval interface{}
			if oval, nval, err = parseAMF0Val(b[n:], offset+n); err != nil {
				err = amf0ParseErr("object.val", offset+n, err)
				return
			}
			n += nval

			obj[okey] = oval
		}
		if len(b) < n+1 {
			err = amf0ParseErr("object.end", offset+n, err)
			return
		}
		n++
		val = obj

	case nullmarker:
	case undefinedmarker:

	case ecmaarraymarker:
		if len(b) < n+4 {
			err = amf0ParseErr("array.count", offset+n, err)
			return
		}
		n += 4

		obj := AMFMap{}
		for {
			if len(b) < n+2 {
				err = amf0ParseErr("array.key.length", offset+n, err)
				return
			}
			length := int(pio.U16BE(b[n:]))
			n += 2

			if length == 0 {
				break
			}

			if len(b) < n+length {
				err = amf0ParseErr("array.key.body", offset+n, err)
				return
			}
			okey := string(b[n:n+length])
			n += length

			var nval int
			var oval interface{}
			if oval, nval, err = parseAMF0Val(b[n:], offset+n); err != nil {
				err = amf0ParseErr("array.val", offset+n, err)
				return
			}
			n += nval

			obj[okey] = oval
		}
		if len(b) < n+1 {
			err = amf0ParseErr("array.end", offset+n, err)
			return
		}
		n += 1
		val = obj

	case objectendmarker:
		if len(b) < n+3 {
			err = amf0ParseErr("objectend", offset+n, err)
			return
		}
		n += 3

	case strictarraymarker:
		if len(b) < n+4 {
			err = amf0ParseErr("strictarray.count", offset+n, err)
			return
		}
		count := int(pio.U32BE(b[n:]))
		n += 4

		obj := make(AMFArray, count)
		for i := 0; i < int(count); i++ {
			var nval int
			if obj[i], nval, err = parseAMF0Val(b[n:], offset+n); err != nil {
				err = amf0ParseErr("strictarray.val", offset+n, err)
				return
			}
			n += nval
		}
		val = obj

	case datemarker:
		if len(b) < n+8+2 {
			err = amf0ParseErr("date", offset+n, err)
			return
		}
		ts := parseBEFloat64(b[n:])
		n += 8+2

		val = time.Unix(int64(ts/1000), (int64(ts)%1000)*1000000)

	case longstringmarker:
		if len(b) < n+4 {
			err = amf0ParseErr("longstring.length", offset+n, err)
			return
		}
		length := int(pio.U32BE(b[n:]))
		n += 4

		if len(b) < n+length {
			err = amf0ParseErr("longstring.body", offset+n, err)
			return
		}
		val = string(b[n:n+length])
		n += length

	default:
		err = amf0ParseErr(fmt.Sprintf("invalidmarker=%d", marker), offset+n, err)
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

