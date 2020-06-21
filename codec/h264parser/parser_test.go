package h264parser

import (
	"encoding/hex"
	"testing"
)

func TestParser(t *testing.T) {
	var ok bool
	var nalus [][]byte

	annexbFrame, _ := hex.DecodeString("00000001223322330000000122332233223300000133000001000001")
	nalus, ok = SplitNALUs(annexbFrame)
	t.Log(ok, len(nalus))

	avccFrame, _ := hex.DecodeString(
		"00000008aabbccaabbccaabb00000001aa",
	)
	nalus, ok = SplitNALUs(avccFrame)
	t.Log(ok, len(nalus))
}
