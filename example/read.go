
package main

import (
	mp4 "./.."
	"log"
)

func main() {
	if _, err := mp4.Open("tiny2-avconv.mp4"); err != nil {
		log.Println(err)
		return
	}
}

