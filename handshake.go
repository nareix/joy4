
package rtmp

import (
	"io"
	"crypto/hmac"
	"crypto/sha256"
	"bytes"
	"math/rand"
)

var (
	clientKey = []byte{
		'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
		'F', 'l', 'a', 's', 'h', ' ', 'P', 'l', 'a', 'y', 'e', 'r', ' ',
		'0', '0', '1',

		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
		0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	serverKey = []byte{
    'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
    'F', 'l', 'a', 's', 'h', ' ', 'M', 'e', 'd', 'i', 'a', ' ',
    'S', 'e', 'r', 'v', 'e', 'r', ' ',
    '0', '0', '1',

    0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
    0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
    0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	clientKey2 = clientKey[:30]
	serverKey2 = serverKey[:36]
	serverVersion = []byte{
    0x0D, 0x0E, 0x0A, 0x0D,
	}
)

func makeDigest(key []byte, src []byte, skip int) (dst []byte) {
	h := hmac.New(sha256.New, key)
	if skip >= 0 && skip < len(src) {
		if skip != 0 {
			h.Write(src[:skip])
		}
		if len(src) != skip + 32 {
			h.Write(src[skip+32:])
		}
	} else {
		h.Write(src)
	}
	return h.Sum(nil)
}

func findDigest(b []byte, key []byte, base int) (int) {
	offs := 0
	for n := 0; n < 4; n++ {
		offs += int(b[base + n])
	}
	offs = (offs % 728) + base + 4
//	fmt.Printf("offs %v\n", offs)
	dig := makeDigest(key, b, offs)
//	fmt.Printf("digest %v\n", digest)
//	fmt.Printf("p %v\n", b[offs:offs+32])
	if bytes.Compare(b[offs:offs+32], dig) != 0 {
		offs = -1
	}
	return offs
}

func writeDigest(b []byte, key []byte, base int) {
	offs := 0
	for n := 8; n < 12; n++ {
		offs += int(b[base + n])
	}
	offs = (offs % 728) + base + 12

	dig := makeDigest(key, b, offs)
	copy(b[offs:], dig)
}

func createChal(b []byte, ver []byte, key []byte) {
	b[0] = 3
	copy(b[5:9], ver)
	for i := 9; i < 1537; i++ {
		b[i] = byte(rand.Int() % 256)
	}
	writeDigest(b[1:], key, 0)
}

func createResp(b []byte, key []byte) {
	for i := 0; i < 1536; i++ {
		b[i] = byte(rand.Int() % 256)
	}
	dig := makeDigest(key, b, 1536-32)
	copy(b[1536-32:], dig)
}

func parseChal(b []byte, peerKey []byte, key []byte) (dig []byte, err int) {
	if b[0] != 0x3 {
		l.Printf("handshake: invalid rtmp version")
		err = 1
		return
	}

	epoch := b[1:5]
	ver := b[5:9]
	l.Printf("handshake:   epoch %v ver %v", epoch, ver)

	var offs int
	if offs = findDigest(b[1:], peerKey, 772); offs == -1 {
		if offs = findDigest(b[1:], peerKey, 8); offs == -1 {
			l.Printf("handshake: digest not found")
			err = 1
			return
		}
	}

	l.Printf("handshake:   offs = %v", offs)

	dig = makeDigest(key, b[1+offs:1+offs+32], -1)
	return
}


func handShake(rw io.ReadWriter) {
	b := ReadBuf(rw, 1537)
	l.Printf("handshake: got client chal")
	dig, err := parseChal(b, clientKey2, serverKey)
	if err != 0 {
		return
	}

	createChal(b, serverVersion, serverKey2)
	l.Printf("handshake: send server chal")
	rw.Write(b)

	b = make([]byte, 1536)
	createResp(b, dig)
	l.Printf("handshake: send server resp")
	rw.Write(b)

	b = ReadBuf(rw, 1536)
	l.Printf("handshake: got client resp")
}

