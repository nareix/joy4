package proxy

import (
	"testing"
	"fmt"
	"time"
	"github.com/nareix/av"
)

func TestProxy(t *testing.T) {
	proxy := New()
	pub, _ := proxy.Publish("abc")
	pub.WriteHeader([]av.CodecData{nil, nil})

	done := make(chan int)

	for n := 0; n < 3; n++ {
		go func(n int) {
			sub, _ := proxy.Subscribe("abc")
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
			time.Sleep(time.Second/100)
		}
		pub.Close()
		done <- 4
	}()

	for i := 0; i < 4; i++ {
		fmt.Println(<-done)
	}

	fmt.Println("complete")

	done = make(chan int)

	proxy = New()
	proxy.HandleSubscribe(func(pub *Publisher) {
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
		sub, _ := proxy.Subscribe("xxoo")
		subs = append(subs, sub)
	}

	for _, sub := range subs {
		sub.Close()
	}

	<-done
}

