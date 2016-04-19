package ts

import (
	"testing"
)

func TestChecksum(t *testing.T) {
	b := []byte("hello world")
	b = append(b, []byte{0xbb, 0x08, 0xec, 0x87}...)
	crc := updateIeeeCrc32(0xffffffff, b)
	t.Logf("%x", crc)
}
