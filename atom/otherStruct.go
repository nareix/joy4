
package atom

import (
	"io"
	"fmt"
	"bytes"
)

type GolombBitReader struct {
	R io.Reader
	buf [1]byte
	left byte
}

func (self *GolombBitReader) ReadBit() (res uint, err error) {
	if self.left == 0 {
		if _, err = self.R.Read(self.buf[:]); err != nil {
			return
		}
		self.left = 8
	}
	self.left--
	res = uint(self.buf[0]>>self.left)&1
	return
}

func (self *GolombBitReader) ReadBits(n int) (res uint, err error) {
	for i := 0; i < n; i++ {
		var bit uint
		if bit, err = self.ReadBit(); err != nil {
			return
		}
		res |= bit<<uint(n-i-1)
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
	res += (1<<uint(i)) - 1
	return
}

func (self *GolombBitReader) ReadSE() (res uint, err error) {
	if res, err = self.ReadExponentialGolombCode(); err != nil {
		return
	}
	if res & 0x01 != 0 {
		res = (res+1)/2
	} else {
		res = -res/2
	}
	return
}

type H264SPSInfo struct {
	ProfileIdc uint
	LevelIdc uint

	MbWidth uint
	MbHeight uint

	CropLeft uint
	CropRight uint
	CropTop uint
	CropBottom uint

	Width uint
	Height uint
}

func ParseH264SPS(data []byte) (res *H264SPSInfo, err error) {
	r := &GolombBitReader{
		R: bytes.NewReader(data),
	}

	self := &H264SPSInfo{}

	if self.ProfileIdc, err = r.ReadBits(8); err != nil {
		return
	}

	// constraint_set0_flag-constraint_set6_flag,reserved_zero_2bits
	if _, err = r.ReadBits(8); err != nil {
		return
	}

	// level_idc
	if self.LevelIdc, err = r.ReadBits(8); err != nil {
		return
	}

	// seq_parameter_set_id
	if _, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}

	if self.ProfileIdc == 100 || self.ProfileIdc == 110 ||
		self.ProfileIdc == 122 || self.ProfileIdc == 244 ||
		self.ProfileIdc == 44 || self.ProfileIdc == 83 ||
		self.ProfileIdc == 86 || self.ProfileIdc == 118 {

		var chroma_format_idc uint
		if chroma_format_idc, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}

		if chroma_format_idc == 3 {
			// residual_colour_transform_flag
			if _, err = r.ReadBit(); err != nil {
				return
			}
		}

		// bit_depth_luma_minus8
		if _, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		// bit_depth_chroma_minus8
		if _, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		// qpprime_y_zero_transform_bypass_flag
		if _, err = r.ReadBit(); err != nil {
			return
		}

		var seq_scaling_matrix_present_flag uint
		if seq_scaling_matrix_present_flag, err = r.ReadBit(); err != nil {
			return
		}

		if seq_scaling_matrix_present_flag != 0 {
			for i := 0; i < 8; i++ {
				var seq_scaling_list_present_flag uint
				if seq_scaling_list_present_flag, err = r.ReadBit(); err != nil {
					return
				}
				if seq_scaling_list_present_flag != 0 {
					var sizeOfScalingList uint
					if i < 6 {
						sizeOfScalingList = 16
					} else {
						sizeOfScalingList = 64
					}
					lastScale := uint(8)
					nextScale := uint(8)
					for j := uint(0); j < sizeOfScalingList; j++ {
						if nextScale != 0 {
							var delta_scale uint
							if delta_scale, err = r.ReadSE(); err != nil {
								return
							}
							nextScale = (lastScale + delta_scale + 256) % 256
						}
						if nextScale != 0 {
							lastScale = nextScale
						}
					}
				}
			}
		}
	}

	// log2_max_frame_num_minus4 
	if _, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}

	var pic_order_cnt_type uint
	if pic_order_cnt_type, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}
	if pic_order_cnt_type == 0 {
		// log2_max_pic_order_cnt_lsb_minus4
		if _, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
	} else if pic_order_cnt_type == 1 {
		// delta_pic_order_always_zero_flag
		if _, err = r.ReadBit(); err != nil {
			return
		}
		// offset_for_non_ref_pic
		if _, err = r.ReadSE(); err != nil {
			return
		}
		// offset_for_top_to_bottom_field
		if _, err = r.ReadSE(); err != nil {
			return
		}
		var num_ref_frames_in_pic_order_cnt_cycle uint
		if num_ref_frames_in_pic_order_cnt_cycle, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		for i := uint(0); i < num_ref_frames_in_pic_order_cnt_cycle; i++ {
			if _, err = r.ReadSE(); err != nil {
				return
			}
		}
	}

	// max_num_ref_frames
	if _, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}

	// gaps_in_frame_num_value_allowed_flag
	if _, err = r.ReadBit(); err != nil {
		return
	}

	if self.MbWidth, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}
	self.MbWidth++

	if self.MbHeight, err = r.ReadExponentialGolombCode(); err != nil {
		return
	}
	self.MbHeight++

	var frame_mbs_only_flag uint
	if frame_mbs_only_flag, err = r.ReadBit(); err != nil {
		return
	}
	if frame_mbs_only_flag == 0 {
		// mb_adaptive_frame_field_flag
		if _, err = r.ReadBit(); err != nil {
			return
		}
	}

	// direct_8x8_inference_flag
	if _, err = r.ReadBit(); err != nil {
		return
	}

	var frame_cropping_flag uint
	if frame_cropping_flag, err = r.ReadBit(); err != nil {
		return
	}
	if frame_cropping_flag != 0 {
		if self.CropLeft, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		if self.CropRight, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		if self.CropTop, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
		if self.CropBottom, err = r.ReadExponentialGolombCode(); err != nil {
			return
		}
	}

	self.Width = (self.MbWidth*16) - self.CropLeft*2 - self.CropRight*2
	self.Height = ((2-frame_mbs_only_flag)*self.MbHeight*16) - self.CropTop*2 - self.CropBottom*2

	res = self
	return
}

type AVCDecoderConfRecord struct {
	AVCProfileIndication int
	ProfileCompatibility int
	AVCLevelIndication int
	LengthSizeMinusOne int
	SPS [][]byte
	PPS [][]byte
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
	SPS []byte,
	PPS []byte,
) (self AVCDecoderConfRecord, err error) {
	if len(SPS) < 4 {
		err = fmt.Errorf("invalid SPS data")
		return
	}
	self.AVCProfileIndication = int(SPS[1])
	self.ProfileCompatibility = int(SPS[2])
	self.AVCLevelIndication = int(SPS[3])
	self.SPS = [][]byte{SPS}
	self.PPS = [][]byte{PPS}
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

	if err = WriteInt(w, len(self.SPS) | 0xe0, 1); err != nil {
		return
	}
	for _, data := range self.SPS {
		if err = WriteInt(w, len(data), 2); err != nil {
			return
		}
		if err = WriteBytes(w, data, len(data)); err != nil {
			return
		}
	}

	if err = WriteInt(w, len(self.PPS), 1); err != nil {
		return
	}
	for _, data := range self.PPS {
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
		self.SPS = append(self.SPS, data)
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
		self.PPS = append(self.PPS, data)
	}

	return
}

