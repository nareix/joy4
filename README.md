Rtmp

Golang rtmp server

Run a simple server 

	package main

	import "github.com/go-av/rtmp"

	func main() {
		rtmp.SimpleServer()
	}

Use avconv to publish stream
	
	avconv -re -i a.mp4 -c:a copy -c:v copy -f flv rtmp://localhost/myapp/1

Use avplay to play stream

	avplay rtmp://localhost/myapp/1

