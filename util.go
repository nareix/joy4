
package av

const (
	H264 = 1
	AAC = 2
)

type Packet struct {
	Codec int
	Key bool
	Pos float32
	Data []byte
	Idx int
}

