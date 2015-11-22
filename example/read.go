
package main

import (
	mp4 "./.."
	"flag"
	"log"
)

func main() {
	testconv := flag.Bool("testconv", false, "")
	flag.Parse()

	if *testconv {
		if _, err := mp4.TestConvert(flag.Arg(0)); err != nil {
			log.Println(err)
			return
		}
	}
}

