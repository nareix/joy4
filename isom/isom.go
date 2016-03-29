package isom

import (
	"github.com/nareix/bits"
	"io"
	"bytes"
	"fmt"
	"io/ioutil"
)

// copied from libavformat/isom.h
const (
	MP4ESDescrTag          = 3
	MP4DecConfigDescrTag   = 4
	MP4DecSpecificDescrTag = 5
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
	SampleRate  int
	ChannelCount int
	ObjectType      uint
	SampleRateIndex uint
	ChannelConfig      uint
}

var sampleRateTable = []int{
	96000, 88200, 64000, 48000, 44100, 32000,
	24000, 22050, 16000, 12000, 11025, 8000, 7350,
}

var chanConfigTable = []int{
	0, 1, 2, 3, 4, 5, 6, 8,
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

// copied from libavcodec/mpeg4audio.c avpriv_mpeg4audio_get_config()
func ReadMPEG4AudioConfig(r io.Reader) (config MPEG4AudioConfig, err error) {
	br := &bits.Reader{R: r}

	if config.ObjectType, err = readObjectType(br); err != nil {
		return
	}
	if config.SampleRateIndex, err = readSampleRateIndex(br); err != nil {
		return
	}
	if int(config.SampleRateIndex) < len(sampleRateTable) {
		config.SampleRate = sampleRateTable[config.SampleRateIndex]
	}
	if config.ChannelConfig, err = br.ReadBits(4); err != nil {
		return
	}
	if int(config.ChannelConfig) < len(chanConfigTable) {
		config.ChannelCount = chanConfigTable[config.ChannelConfig]
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

func readDesc(r io.Reader) (tag uint, data []byte, err error) {
	if tag, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	var length uint
	for i := 0; i < 4; i++ {
		var c uint
		if c, err = bits.ReadUIntBE(r, 8); err != nil {
			return
		}
		length = (length << 7) | (c & 0x7f)
		if c&0x80 == 0 {
			break
		}
	}
	data = make([]byte, length)
	if _, err = r.Read(data); err != nil {
		return
	}
	return
}

func writeDesc(w io.Writer, tag uint, data []byte) (err error) {
	if err = bits.WriteUIntBE(w, tag, 8); err != nil {
		return
	}
	length := uint(len(data))
	for length > 0 {
		val := length&0x7f
		if length >= 0x80 {
			val |= 0x80
		}
		if err = bits.WriteUIntBE(w, val, 8); err != nil {
			return
		}
		length >>= 7
	}
	if _, err = w.Write(data); err != nil {
		return
	}
	return
}

func readESDesc(r io.Reader) (err error) {
	// ES_ID
	if _, err = bits.ReadUIntBE(r, 16); err != nil {
		return
	}
	var flags uint
	if flags, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	//streamDependenceFlag
	if flags&0x80 != 0 {
		if _, err = bits.ReadUIntBE(r, 16); err != nil {
			return
		}
	}
	//URL_Flag
	if flags&0x40 != 0 {
		var length uint
		if length, err = bits.ReadUIntBE(r, 8); err != nil {
			return
		}
		if _, err = io.CopyN(ioutil.Discard, r, int64(length)); err != nil {
			return
		}
	}
	//OCRstreamFlag
	if flags&0x20 != 0 {
		if _, err = bits.ReadUIntBE(r, 16); err != nil {
			return
		}
	}
	return
}

func writeESDesc(w io.Writer) (err error) {
	// ES_ID
	if err = bits.WriteUIntBE(w, 0, 16); err != nil {
		return
	}
	// flags
	if err = bits.WriteUIntBE(w, 0, 8); err != nil {
		return
	}
	return
}

// copied from libavformat/isom.c ff_mp4_read_dec_config_descr()
func readDecConfDesc(r io.Reader) (decConfig []byte, err error) {
	var objectId uint
	var streamType uint
	var bufSize uint
	var maxBitrate uint
	var avgBitrate uint

	// objectId
	if objectId, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	// streamType
	if streamType, err = bits.ReadUIntBE(r, 8); err != nil {
		return
	}
	// buffer size db
	if bufSize, err = bits.ReadUIntBE(r, 24); err != nil {
		return
	}
	// max bitrate
	if maxBitrate, err = bits.ReadUIntBE(r, 32); err != nil {
		return
	}
	// avg bitrate
	if avgBitrate, err = bits.ReadUIntBE(r, 32); err != nil {
		return
	}

	if false {
		println("readDecConfDesc", objectId, streamType, bufSize, maxBitrate, avgBitrate)
	}

	var tag uint
	var data []byte
	if tag, data, err = readDesc(r); err != nil {
		return
	}
	if tag != MP4DecSpecificDescrTag {
		err = fmt.Errorf("MP4DecSpecificDescrTag not found")
		return
	}
	decConfig = data
	return
}

// copied from libavformat/movenc.c mov_write_esds_tag()
func writeDecConfDesc(w io.Writer, objectId uint, streamType uint, decConfig []byte) (err error) {
	// objectId
	if err = bits.WriteUIntBE(w, objectId, 8); err != nil {
		return
	}
	// streamType
	if err = bits.WriteUIntBE(w, streamType, 8); err != nil {
		return
	}
	// buffer size db
	if err = bits.WriteUIntBE(w, 0, 24); err != nil {
		return
	}
	// max bitrate
	if err = bits.WriteUIntBE(w, 0, 32); err != nil {
		return
	}
	// avg bitrate
	if err = bits.WriteUIntBE(w, 0, 32); err != nil {
		return
	}
	if err = writeDesc(w, MP4DecSpecificDescrTag, decConfig); err != nil {
		return
	}
	return
}

// copied from libavformat/mov.c ff_mov_read_esds()
func ReadElemStreamDesc(r io.Reader) (decConfig []byte, err error) {
	var tag uint
	var data []byte
	if tag, data, err = readDesc(r); err != nil {
		return
	}
	if tag != MP4ESDescrTag {
		err = fmt.Errorf("MP4ESDescrTag not found")
		return
	}
	r = bytes.NewReader(data)

	if err = readESDesc(r); err != nil {
		return
	}
	if tag, data, err = readDesc(r); err != nil {
		return
	}
	if tag != MP4DecConfigDescrTag {
		err = fmt.Errorf("MP4DecSpecificDescrTag not found")
		return
	}
	r = bytes.NewReader(data)

	if decConfig, err = readDecConfDesc(r); err != nil {
		return
	}
	return
}

func WriteElemStreamDescAAC(w io.Writer, config MPEG4AudioConfig) (err error) {
	// MP4ESDescrTag(ESDesc MP4DecConfigDescrTag(objectId streamType bufSize avgBitrate MP4DecSpecificDescrTag(decConfig)))

	buf := &bytes.Buffer{}
	WriteMPEG4AudioConfig(buf, config)
	data := buf.Bytes()

	buf = &bytes.Buffer{}
	// 0x40 = ObjectType AAC
	// 0x15 = Audiostream
	writeDecConfDesc(buf, 0x40, 0x15, data)
	data = buf.Bytes()

	buf = &bytes.Buffer{}
	writeDesc(buf, MP4DecConfigDescrTag, data)
	data = buf.Bytes()

	buf = &bytes.Buffer{}
	writeESDesc(buf)
	buf.Write(data)
	data = buf.Bytes()

	buf = &bytes.Buffer{}
	writeDesc(buf, MP4ESDescrTag, data)
	data = buf.Bytes()

	if _, err = w.Write(data); err != nil {
		return
	}

	return
}

