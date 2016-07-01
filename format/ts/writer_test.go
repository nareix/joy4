package ts

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestWriteTSHeader(t *testing.T) {
	bw := &bytes.Buffer{}
	w := &TSWriter{
		W:   bw,
		PCR: 0x12345678,
	}
	w.Write([]byte{'h', 'e', 'l', 'o'}[:], false)
	t.Logf("\n%s", hex.Dump(bw.Bytes()))
}
