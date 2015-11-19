
package mp4

type Mp4a struct {
	Config []byte
	SampleRate int
	Channels int
}

// [dur][dur][dur][dur]

// Read()
// Write(i, buf, dur)
// RemoveAll()

