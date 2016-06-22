package flvio

import (
	"math"
	"time"
	"github.com/nareix/pio"
)

type AMFMap map[string]interface{}
type AMFArray []interface{}

func readBEFloat64(r *pio.Reader) (f64 float64, err error) {
	var u64 uint64
	if u64, err = r.ReadU64BE(); err != nil {
		return
	}
	f64 = math.Float64frombits(u64)
	return
}

func ReadAMF0Val(r *pio.Reader) (val interface{}, err error) {
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
	)

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
	}

	return
}

