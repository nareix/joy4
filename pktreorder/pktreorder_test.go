package pktreorder

import (
	"github.com/nareix/av"
	"fmt"
)

type fakeStream struct {
	isVideo bool
	type_ int
}

func (self fakeStream) IsVideo() bool {
	return self.isVideo
}

func (self fakeStream) IsAudio() bool {
	return !self.isVideo
}

func (self fakeStream) Type() int {
	return self.type_
}

func ExampleQueue() {
	var streams []av.CodecData
	streams = append(streams, fakeStream{isVideo: true})
	streams = append(streams, fakeStream{isVideo: false})

	queue := &Queue{}
	queue.Alloc(streams)
	var i int
	var err error

	/*
	Output:
false
false
true
0 true
1 true
1 true
1 true
0 false
0.30
1 true
0 true
1 true
1 true
0 true
0 true
0 false
0 false
	*/

	fmt.Println(queue.CanReadPacket())
	queue.WritePacket(1, av.Packet{Duration: 0.1})
	queue.WritePacket(1, av.Packet{Duration: 0.1})
	queue.WritePacket(1, av.Packet{Duration: 0.1})
	fmt.Println(queue.CanReadPacket())

	queue.WritePacket(0, av.Packet{Duration: 1.0})
	queue.WritePacket(0, av.Packet{Duration: 1.0})
	fmt.Println(queue.CanReadPacket())
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)

	queue.WritePacket(1, av.Packet{Duration: 0.8})
	fmt.Println(fmt.Sprintf("%.2f", queue.streams[1].pos))
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)

	queue.WritePacket(0, av.Packet{Duration: 0.1})
	queue.WritePacket(1, av.Packet{Duration: 0.1})
	queue.WritePacket(0, av.Packet{Duration: 0.1})
	queue.WritePacket(1, av.Packet{Duration: 0.1})
	queue.EndWritePacket(nil)

	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
	i, _, err = queue.ReadPacket()
	fmt.Println(i, err == nil)
}

