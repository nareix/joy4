
package main

import (
	mp4 "./.."
	"flag"
	"log"
)

func main() {
	testconv := flag.Bool("testconv", false, "")
	testrewrite := flag.Bool("testrewrite", false, "")
	flag.Parse()

	if *testconv {
		if _, err := mp4.TestConvert(flag.Arg(0)); err != nil {
			log.Println(err)
			return
		}
	}

	if *testrewrite {
		if _, err := mp4.TestRewrite(flag.Arg(0)); err != nil {
			log.Println(err)
			return
		}
	}

}

