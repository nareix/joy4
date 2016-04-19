package ts

import (
	"testing"
)

func TestPESTsConv(t *testing.T) {
	t.Logf("%x", PESTsToUInt(0x123))
}
