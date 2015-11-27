
package main

import (
	mp4 "./.."
	"flag"
	"log"
)

func main() {
	testconv := flag.Bool("testconv", false, "")
	probe := flag.Bool("probe", false, "")
	flag.Parse()

	if *testconv {
		if err := mp4.TestConvert(flag.Arg(0)); err != nil {
			log.Println(err)
			return
		}
	}

	if *probe {
		if err := mp4.ProbeFile(flag.Arg(0)); err != nil {
			log.Println(err)
			return
		}
	}

}

