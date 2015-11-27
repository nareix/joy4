
package atom

import (
	"io"
	"fmt"
)

type AVCDecoderConfRecord struct {
	AVCProfileIndication int
	ProfileCompatibility int
	AVCLevelIndication int
	LengthSizeMinusOne int
	SeqenceParamSet [][]byte
	PictureParamSet [][]byte
}

func WriteSampleByNALU(w io.Writer, nalu []byte) (size int, err error) {
	if err = WriteInt(w, len(nalu), 4); err != nil {
		return
	}
	size += 4
	if _, err = w.Write(nalu); err != nil {
		return
	}
	size += len(nalu)
	return
}

func CreateAVCDecoderConfRecord(
	SeqenceParamSet []byte,
	PictureParamSet []byte,
) (self AVCDecoderConfRecord, err error) {
	if len(SeqenceParamSet) < 4 {
		err = fmt.Errorf("invalid SeqenceParamSet data")
		return
	}
	self.AVCProfileIndication = int(SeqenceParamSet[1])
	self.AVCLevelIndication = int(SeqenceParamSet[3])
	self.SeqenceParamSet = [][]byte{SeqenceParamSet}
	self.PictureParamSet = [][]byte{PictureParamSet}
	self.LengthSizeMinusOne = 3
	return
}

func WriteAVCDecoderConfRecord(w io.Writer, self AVCDecoderConfRecord) (err error) {
	if err = WriteInt(w, 1, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.AVCProfileIndication, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.ProfileCompatibility, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.AVCLevelIndication, 1); err != nil {
		return
	}
	if err = WriteInt(w, self.LengthSizeMinusOne | 0xfc, 1); err != nil {
		return
	}

	if err = WriteInt(w, len(self.SeqenceParamSet) | 0xe0, 1); err != nil {
		return
	}
	for _, data := range self.SeqenceParamSet {
		if err = WriteInt(w, len(data), 2); err != nil {
			return
		}
		if err = WriteBytes(w, data, len(data)); err != nil {
			return
		}
	}

	if err = WriteInt(w, len(self.PictureParamSet), 1); err != nil {
		return
	}
	for _, data := range self.PictureParamSet {
		if err = WriteInt(w, len(data), 2); err != nil {
			return
		}
		if err = WriteBytes(w, data, len(data)); err != nil {
			return
		}
	}

	return
}

func WalkAVCDecoderConfRecord(w Walker, self AVCDecoderConfRecord) {
}

func ReadAVCDecoderConfRecord(r *io.LimitedReader) (self AVCDecoderConfRecord, err error) {
	if _, err = ReadDummy(r, 1); err != nil {
		return
	}
	if self.AVCProfileIndication, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.ProfileCompatibility, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.AVCLevelIndication, err = ReadInt(r, 1); err != nil {
		return
	}
	if self.LengthSizeMinusOne, err = ReadInt(r, 1); err != nil {
		return
	}
	self.LengthSizeMinusOne &= 0x03

	var n, length int
	var data []byte

	if n, err = ReadInt(r, 1); err != nil {
		return
	}
	n &= 0x1f
	for i := 0; i < n; i++ {
		if length, err = ReadInt(r, 2); err != nil {
			return
		}
		if data, err = ReadBytes(r, length); err != nil {
			return
		}
		self.SeqenceParamSet = append(self.SeqenceParamSet, data)
	}

	if n, err = ReadInt(r, 1); err != nil {
		return
	}
	for i := 0; i < n; i++ {
		if length, err = ReadInt(r, 2); err != nil {
			return
		}
		if data, err = ReadBytes(r, length); err != nil {
			return
		}
		self.PictureParamSet = append(self.PictureParamSet, data)
	}

	return
}

