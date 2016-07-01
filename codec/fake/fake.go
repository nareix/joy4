package fake

import (
	"github.com/nareix/av"
)

type CodecData struct {
	Typ av.CodecType
}

func (self CodecData) Type() av.CodecType {
	return self.Typ
}

