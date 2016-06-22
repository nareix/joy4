package channels

import (
	"fmt"
	"github.com/nareix/av"
	"time"
)

func ExampleChannels() {
	/* Output:
	complete
	[xxoo]
	*/
	context := New()
	pub, _ := context.Publish("abc")
	pub.WriteHeader([]av.CodecData{nil, nil})

	done := make(chan int)

	for n := 0; n < 3; n++ {
		go func(n int) {
			sub, _ := context.Subscribe("abc")
			if sub == nil {
				done <- n
				return
			}
			for {
				i, pkt, err := sub.ReadPacket()
				if err != nil {
					break
				}
				fmt.Println(i, pkt)
			}
			fmt.Println("close", n)
			sub.Close()
			done <- n
		}(n)
	}

	go func() {
		pub.WritePacket(1, av.Packet{})
		pub.WritePacket(2, av.Packet{})
		pub.WritePacket(3, av.Packet{})
		if false {
			time.Sleep(time.Second / 100)
		}
		pub.Close()
		done <- 4
	}()

	for i := 0; i < 4; i++ {
		<-done
	}

	fmt.Println("complete")

	done = make(chan int)

	context = New()
	context.HandleSubscribe(func(pub *Publisher) {
		fmt.Println(pub.Params)
		pub.WriteHeader([]av.CodecData{nil, nil})
		for {
			if err := pub.WritePacket(0, av.Packet{}); err != nil {
				break
			}
		}
		done <- 1
	})

	subs := []*Subscriber{}
	for i := 0; i < 3; i++ {
		sub, _ := context.Subscribe("xxoo")
		subs = append(subs, sub)
	}

	for _, sub := range subs {
		sub.Close()
	}

	<-done
}
