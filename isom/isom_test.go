
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

	//00000000  ff f1 50 80 04 3f fc de  04 00 00 6c 69 62 66 61  |..P..?.....libfa|
	//00000010  61 63 20 31 2e 32 38 00  00 42 40 93 20 04 32 00  |ac 1.28..B@. .2.|
	//00000020  47 ff f1 50 80 05 1f fc  21 42 fe ed b2 5c a8 00  |G..P....!B...\..|
	data, _ = hex.DecodeString("fff15080043ffcde040000")
	objectType, sampleRateIndex, chanConfig, frameLength := ReadADTSHeader(data)
	t.Logf("objectType=%d sampleRateIndex=%d chanConfig=%d frameLength=%d", objectType, sampleRateIndex, chanConfig, frameLength)
}

