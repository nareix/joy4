package aacparser

import (
	"github.com/nareix/bits"
	"github.com/nareix/av"
	"fmt"
	"bytes"
	"io"
)

// copied from libavcodec/mpeg4audio.h
const (
	AOT_AAC_MAIN        = 1 + iota  ///< Y                       Main
	AOT_AAC_LC                      ///< Y                       Low Complexity
	AOT_AAC_SSR                     ///< N (code in SoC repo)    Scalable Sample Rate
	AOT_AAC_LTP                     ///< Y                       Long Term Prediction
	AOT_SBR                         ///< Y                       Spectral Band Replication
	AOT_AAC_SCALABLE                ///< N                       Scalable
	AOT_TWINVQ                      ///< N                       Twin Vector Quantizer
	AOT_CELP                        ///< N                       Code Excited Linear Prediction
	AOT_HVXC                        ///< N                       Harmonic Vector eXcitation Coding
	AOT_TTSI            = 12 + iota ///< N                       Text-To-Speech Interface
	AOT_MAINSYNTH                   ///< N                       Main Synthesis
	AOT_WAVESYNTH                   ///< N                       Wavetable Synthesis
	AOT_MIDI                        ///< N                       General MIDI
	AOT_SAFX                        ///< N                       Algorithmic Synthesis and Audio Effects
	AOT_ER_AAC_LC                   ///< N                       Error Resilient Low Complexity
	AOT_ER_AAC_LTP      = 19 + iota ///< N                       Error Resilient Long Term Prediction
	AOT_ER_AAC_SCALABLE             ///< N                       Error Resilient Scalable
	AOT_ER_TWINVQ                   ///< N                       Error Resilient Twin Vector Quantizer
	AOT_ER_BSAC                     ///< N                       Error Resilient Bit-Sliced Arithmetic Coding
	AOT_ER_AAC_LD                   ///< N                       Error Resilient Low Delay
	AOT_ER_CELP                     ///< N                       Error Resilient Code Excited Linear Prediction
	AOT_ER_HVXC                     ///< N                       Error Resilient Harmonic Vector eXcitation Coding
	AOT_ER_HILN                     ///< N                       Error Resilient Harmonic and Individual Lines plus Noise
	AOT_ER_PARAM                    ///< N                       Error Resilient Parametric
	AOT_SSC                         ///< N                       SinuSoidal Coding
	AOT_PS                          ///< N                       Parametric Stereo
	AOT_SURROUND                    ///< N                       MPEG Surround
	AOT_ESCAPE                      ///< Y                       Escape Value
	AOT_L1                          ///< Y                       Layer 1
	AOT_L2                          ///< Y                       Layer 2
	AOT_L3                          ///< Y                       Layer 3
	AOT_DST                         ///< N                       Direct Stream Transfer
	AOT_ALS                         ///< Y                       Audio LosslesS
	AOT_SLS                         ///< N                       Scalable LosslesS
	AOT_SLS_NON_CORE                ///< N                       Scalable LosslesS (non core)
	AOT_ER_AAC_ELD                  ///< N                       Error Resilient Enhanced Low Delay
	AOT_SMR_SIMPLE                  ///< N                       Symbolic Music Representation Simple
	AOT_SMR_MAIN                    ///< N                       Symbolic Music Representation Main
	AOT_USAC_NOSBR                  ///< N                       Unified Speech and Audio Coding (no SBR)
	AOT_SAOC                        ///< N                       Spatial Audio Object Coding
	AOT_LD_SURROUND                 ///< N                       Low Delay MPEG Surround
	AOT_USAC                        ///< N                       Unified Speech and Audio Coding
)

type MPEG4AudioConfig struct {
	SampleRate      int
	ChannelCount    int
	ObjectType      uint
	SampleRateIndex uint
	ChannelConfig   uint
}

var sampleRateTable = []int{
	96000, 88200, 64000, 48000, 44100, 32000,
	24000, 22050, 16000, 12000, 11025, 8000, 7350,
}

var chanConfigTable = []int{
	0, 1, 2, 3, 4, 5, 6, 8,
}

func IsADTSFrame(frames []byte) bool {
	return len(frames) > 7 && frames[0] == 0xff && frames[1]&0xf0 == 0xf0
}

func ReadADTSFrame(frame []byte) (config MPEG4AudioConfig, payload []byte, samples int, framelen int, err error) {
	if !IsADTSFrame(frame) {
		err = fmt.Errorf("not adts frame")
		return
	}
	config.ObjectType = uint(frame[2]>>6) + 1
	config.SampleRateIndex = uint(frame[2] >> 2 & 0xf)
	config.ChannelConfig = uint(frame[2]<<2&0x4 | frame[3]>>6&0x3)
	framelen = int(frame[3]&0x3)<<11 | int(frame[4])<<3 | int(frame[5]>>5)
	samples = (int(frame[6]&0x3) + 1) * 1024
	hdrlen := 7
	if frame[1]&0x1 == 0 {
		hdrlen = 9
	}
	if framelen < hdrlen || len(frame) < framelen {
		err = fmt.Errorf("invalid adts header length")
		return
	}
	payload = frame[hdrlen:framelen]
	return
}

func MakeADTSHeader(config MPEG4AudioConfig, samples int, payloadLength int) (header []byte) {
	payloadLength += 7
	//AAAAAAAA AAAABCCD EEFFFFGH HHIJKLMM MMMMMMMM MMMOOOOO OOOOOOPP (QQQQQQQQ QQQQQQQQ)
	header = []byte{0xff, 0xf1, 0x50, 0x80, 0x043, 0xff, 0xcd}
	//config.ObjectType = uint(frames[2]>>6)+1
	//config.SampleRateIndex = uint(frames[2]>>2&0xf)
	//config.ChannelConfig = uint(frames[2]<<2&0x4|frames[3]>>6&0x3)
	header[2] = (byte(config.ObjectType-1)&0x3)<<6 | (byte(config.SampleRateIndex)&0xf)<<2 | byte(config.ChannelConfig>>2)&0x1
	header[3] = header[3]&0x3f | byte(config.ChannelConfig&0x3)<<6
	header[3] = header[3]&0xfc | byte(payloadLength>>11)&0x3
	header[4] = byte(payloadLength >> 3)
	header[5] = header[5]&0x1f | (byte(payloadLength)&0x7)<<5
	header[6] = header[6]&0xfc | byte(samples/1024-1)
	return
}

func ExtractADTSFrames(frames []byte) (config MPEG4AudioConfig, payload [][]byte, samples int, err error) {
	for len(frames) > 0 {
		var n, framelen int
		var _payload []byte
		if config, _payload, n, framelen, err = ReadADTSFrame(frames); err != nil {
			return
		}
		payload = append(payload, _payload)
		frames = frames[framelen:]
		samples += n
	}
	return
}

func ReadADTSHeader(data []byte) (config MPEG4AudioConfig, frameLength int) {
	br := &bits.Reader{R: bytes.NewReader(data)}
	var i uint

	//Structure
	//AAAAAAAA AAAABCCD EEFFFFGH HHIJKLMM MMMMMMMM MMMOOOOO OOOOOOPP (QQQQQQQQ QQQQQQQQ)
	//Header consists of 7 or 9 bytes (without or with CRC).

	// 2 bytes
	//A	12	syncword 0xFFF, all bits must be 1
	br.ReadBits(12)
	//B	1	MPEG Version: 0 for MPEG-4, 1 for MPEG-2
	br.ReadBits(1)
	//C	2	Layer: always 0
	br.ReadBits(2)
	//D	1	protection absent, Warning, set to 1 if there is no CRC and 0 if there is CRC
	br.ReadBits(1)

	//E	2	profile, the MPEG-4 Audio Object Type minus 1
	config.ObjectType, _ = br.ReadBits(2)
	config.ObjectType++
	//F	4	MPEG-4 Sampling Frequency Index (15 is forbidden)
	config.SampleRateIndex, _ = br.ReadBits(4)
	//G	1	private bit, guaranteed never to be used by MPEG, set to 0 when encoding, ignore when decoding
	br.ReadBits(1)
	//H	3	MPEG-4 Channel Configuration (in the case of 0, the channel configuration is sent via an inband PCE)
	config.ChannelConfig, _ = br.ReadBits(3)
	//I	1	originality, set to 0 when encoding, ignore when decoding
	br.ReadBits(1)
	//J	1	home, set to 0 when encoding, ignore when decoding
	br.ReadBits(1)
	//K	1	copyrighted id bit, the next bit of a centrally registered copyright identifier, set to 0 when encoding, ignore when decoding
	br.ReadBits(1)
	//L	1	copyright id start, signals that this frame's copyright id bit is the first bit of the copyright id, set to 0 when encoding, ignore when decoding
	br.ReadBits(1)

	//M	13	frame length, this value must include 7 or 9 bytes of header length: FrameLength = (ProtectionAbsent == 1 ? 7 : 9) + size(AACFrame)
	i, _ = br.ReadBits(13)
	frameLength = int(i)
	//O	11	Buffer fullness
	br.ReadBits(11)
	//P	2	Number of AAC frames (RDBs) in ADTS frame minus 1, for maximum compatibility always use 1 AAC frame per ADTS frame
	br.ReadBits(2)

	//Q	16	CRC if protection absent is 0
	return
}

func readObjectType(r *bits.Reader) (objectType uint, err error) {
	if objectType, err = r.ReadBits(5); err != nil {
		return
	}
	if objectType == AOT_ESCAPE {
		var i uint
		if i, err = r.ReadBits(6); err != nil {
			return
		}
		objectType = 32 + i
	}
	return
}

func writeObjectType(w *bits.Writer, objectType uint) (err error) {
	if objectType >= 32 {
		if err = w.WriteBits(AOT_ESCAPE, 5); err != nil {
			return
		}
		if err = w.WriteBits(objectType-32, 6); err != nil {
			return
		}
	} else {
		if err = w.WriteBits(objectType, 5); err != nil {
			return
		}
	}
	return
}

func readSampleRateIndex(r *bits.Reader) (index uint, err error) {
	if index, err = r.ReadBits(4); err != nil {
		return
	}
	if index == 0xf {
		if index, err = r.ReadBits(24); err != nil {
			return
		}
	}
	return
}

func writeSampleRateIndex(w *bits.Writer, index uint) (err error) {
	if index >= 0xf {
		if err = w.WriteBits(0xf, 4); err != nil {
			return
		}
		if err = w.WriteBits(index, 24); err != nil {
			return
		}
	} else {
		if err = w.WriteBits(index, 4); err != nil {
			return
		}
	}
	return
}

func (self MPEG4AudioConfig) IsValid() bool {
	return self.ObjectType > 0
}

func (self MPEG4AudioConfig) Complete() (config MPEG4AudioConfig) {
	config = self
	if int(config.SampleRateIndex) < len(sampleRateTable) {
		config.SampleRate = sampleRateTable[config.SampleRateIndex]
	}
	if int(config.ChannelConfig) < len(chanConfigTable) {
		config.ChannelCount = chanConfigTable[config.ChannelConfig]
	}
	return
}

func ParseMPEG4AudioConfig(data []byte) (config MPEG4AudioConfig, err error) {
	r := bytes.NewReader(data)
	if config, err = ReadMPEG4AudioConfig(r); err != nil {
		err = fmt.Errorf("CodecData invalid: parse MPEG4AudioConfig failed(%s)", err)
		return
	}
	config = config.Complete()
	return
}

func ReadMPEG4AudioConfig(r io.Reader) (config MPEG4AudioConfig, err error) {
	// copied from libavcodec/mpeg4audio.c avpriv_mpeg4audio_get_config()
	br := &bits.Reader{R: r}

	if config.ObjectType, err = readObjectType(br); err != nil {
		return
	}
	if config.SampleRateIndex, err = readSampleRateIndex(br); err != nil {
		return
	}
	if config.ChannelConfig, err = br.ReadBits(4); err != nil {
		return
	}
	return
}

func WriteMPEG4AudioConfig(w io.Writer, config MPEG4AudioConfig) (err error) {
	bw := &bits.Writer{W: w}

	if err = writeObjectType(bw, config.ObjectType); err != nil {
		return
	}

	if config.SampleRateIndex == 0 {
		for i, rate := range sampleRateTable {
			if rate == config.SampleRate {
				config.SampleRateIndex = uint(i)
			}
		}
	}
	if err = writeSampleRateIndex(bw, config.SampleRateIndex); err != nil {
		return
	}

	if config.ChannelConfig == 0 {
		for i, count := range chanConfigTable {
			if count == config.ChannelCount {
				config.ChannelConfig = uint(i)
			}
		}
	}
	if err = bw.WriteBits(config.ChannelConfig, 4); err != nil {
		return
	}

	if err = bw.FlushBits(); err != nil {
		return
	}
	return
}

type CodecData struct {
	Config []byte
	ConfigInfo MPEG4AudioConfig
}

func (self CodecData) IsVideo() bool {
	return false
}

func (self CodecData) IsAudio() bool {
	return true
}

func (self CodecData) Type() int {
	return av.AAC
}

func (self CodecData) MPEG4AudioConfigBytes() []byte {
	return self.Config
}

func (self CodecData) ChannelCount() int {
	return self.ConfigInfo.ChannelCount
}

func (self CodecData) SampleRate() int {
	return self.ConfigInfo.SampleRate
}

func (self CodecData) SampleFormat() av.SampleFormat {
	return av.FLTP
}

func (self CodecData) MakeADTSHeader(samples int, payloadLength int) []byte {
	return MakeADTSHeader(self.ConfigInfo, samples, payloadLength)
}

func NewCodecDataFromMPEG4AudioConfigBytes(config []byte) (codec av.AACCodecData, err error) {
	self := CodecData{}
	self.Config = config
	if self.ConfigInfo, err = ParseMPEG4AudioConfig(config); err != nil {
		err = fmt.Errorf("parse MPEG4AudioConfig failed(%s)", err)
		return
	}
	codec = self
	return
}

