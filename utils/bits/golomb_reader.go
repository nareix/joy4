package bits

import (
	"io"
)

type GolombBitReader struct {
	R    io.Reader
	buf  [1]byte
	left byte
	prev_two_bytes uint
	emulation_prevention_bytes uint
}

func (self *GolombBitReader) ReadBit() (res uint, err error) {
	if self.left == 0 {
		if _, err = self.R.Read(self.buf[:]); err != nil {
			return
		}
		/*
		// Emulation prevention three-byte detection.
		// If a sequence of 0x000003 is found, skip (ignore) the last byte (0x03).
		*/
		if self.buf[0] == 0x03 && (self.prev_two_bytes & 0xffff) == 0 {
			// Detected 0x000003, skip last byte.
			if _, err = self.R.Read(self.buf[:]); err != nil {
				return
			}

			self.emulation_prevention_bytes++
			/*
			// Need another full three bytes before we can detect the sequence again.
			*/
			self.prev_two_bytes = 0xffff
		}
		self.left = 8
		self.prev_two_bytes = (self.prev_two_bytes << 8) | uint(self.buf[0])
	}
	self.left--
	res = uint(self.buf[0]>>self.left) & 1
	return
}

func (self *GolombBitReader) ReadBits(n int) (res uint, err error) {
	for i := 0; i < n; i++ {
		var bit uint
		if bit, err = self.ReadBit(); err != nil {
			return
		}
		res |= bit << uint(n-i-1)
	}
	return
}

func (self *GolombBitReader) ReadExponentialGolombCode() (res uint, err error) {
	i := 0
	for {
		var bit uint
		if bit, err = self.ReadBit(); err != nil {
			return
		}
		if !(bit == 0 && i < 32) {
			break
		}
		i++
	}
	if res, err = self.ReadBits(i); err != nil {
		return
	}
	res += (1 << uint(i)) - 1
	return
}

func (self *GolombBitReader) ReadSE() (res uint, err error) {
	if res, err = self.ReadExponentialGolombCode(); err != nil {
		return
	}
	if res&0x01 != 0 {
		res = (res + 1) / 2
	} else {
		res = -res / 2
	}
	return
}

func (self *GolombBitReader) NumEmulationPreventionBytesRead() uint {
	return self.emulation_prevention_bytes
}
