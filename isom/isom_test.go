package isom

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestReadElemStreamDesc(t *testing.T) {
	debugReader = true
	debugWriter = true

	var err error

	data, _ := hex.DecodeString("03808080220002000480808014401500000000030d400000000005808080021210068080800102")
	t.Logf("elemDesc=%x", data)
	t.Logf("length=%d", len(data))

	var aconfig MPEG4AudioConfig
	if aconfig, err = ReadElemStreamDescAAC(bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	aconfig = aconfig.Complete()
	t.Logf("aconfig=%v", aconfig)

	bw := &bytes.Buffer{}
	WriteMPEG4AudioConfig(bw, aconfig)

	bw = &bytes.Buffer{}
	WriteElemStreamDescAAC(bw, aconfig, 2)
	t.Logf("elemDesc=%x", bw.Bytes())
	data = bw.Bytes()
	t.Logf("length=%d", len(data))

	if aconfig, err = ReadElemStreamDescAAC(bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	t.Logf("aconfig=%v", aconfig.Complete())

	//00000000  ff f1 50 80 04 3f fc de  04 00 00 6c 69 62 66 61  |..P..?.....libfa|
	//00000010  61 63 20 31 2e 32 38 00  00 42 40 93 20 04 32 00  |ac 1.28..B@. .2.|
	//00000020  47 ff f1 50 80 05 1f fc  21 42 fe ed b2 5c a8 00  |G..P....!B...\..|
	data, _ = hex.DecodeString("fff15080043ffcde040000")
	var n, framelen int
	aconfig, _, n, _, _ = ReadADTSFrame(data)
	t.Logf("%v n=%d", aconfig.Complete(), n)

	data = MakeADTSHeader(aconfig, 1024*3, 33)
	data = append(data, []byte{1,2,3,4,5}...)
	t.Logf("%x", data)
	aconfig, _, n, framelen, err = ReadADTSFrame(data)
	t.Logf("%v n=%d framelen=%d err=%v", aconfig.Complete(), n, framelen, err)
}
