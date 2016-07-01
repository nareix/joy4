package fake

import (
	"github.com/nareix/joy4/av"
)

type CodecData struct {
	Typ av.CodecType
}

func (self CodecData) Type() av.CodecType {
	return self.Typ
}

