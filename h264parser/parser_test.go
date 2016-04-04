
package h264parser

import (
	"testing"
	"encoding/hex"
)

func TestParser(t *testing.T) {
	var ok bool
	var nalus [][]byte

	annexbFrame, _ := hex.DecodeString("000001223322330000000122332233223300000133000001000001")
	ok, nalus = SplitNALUs(annexbFrame)
	t.Log(ok, len(nalus))

	avccFrame, _ := hex.DecodeString(
		"00000008aabbccaabbccaabb00000001aa",
	)
	ok, nalus = SplitNALUs(avccFrame)
	t.Log(ok, len(nalus))
}

