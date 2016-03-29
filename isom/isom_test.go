
package isom

import (
	"testing"
	"encoding/hex"
	"bytes"
)

func TestReadElemStreamDesc(t *testing.T) {
	var decConfig []byte
	var err error

	data, _ := hex.DecodeString("03808080220002000480808014401500000000030d400000000005808080021210068080800102")
	t.Logf("length=%d", len(data))

	if decConfig, err = ReadElemStreamDesc(bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	t.Logf("decConfig=%x", decConfig)

	var aconfig MPEG4AudioConfig
	if aconfig, err = ReadMPEG4AudioConfig(bytes.NewReader(decConfig)); err != nil {
		t.Error(err)
	}
	t.Logf("aconfig=%v", aconfig)

	bw := &bytes.Buffer{}
	WriteMPEG4AudioConfig(bw, aconfig)
	t.Logf("decConfig=%x", bw.Bytes())

	bw = &bytes.Buffer{}
	WriteElemStreamDescAAC(bw, aconfig)
	t.Logf("elemDesc=%x", bw.Bytes())
	data = bw.Bytes()

	if decConfig, err = ReadElemStreamDesc(bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	t.Logf("decConfig=%x", decConfig)
}

