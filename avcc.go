
package mp4

type Avcc struct {
	Config []byte
	W, H int
	Fps int
}

// [dur][dur][dur][dur]

// Duration() dur
// FindKeyFrame(at) at, dur
// Read(at) at, dur, buf
// Write(at, buf, dur) at
// RemoveAll()

