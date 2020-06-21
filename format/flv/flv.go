package flv

import (
	"bufio"
	"fmt"
	"io"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/codec"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/joy4/codec/fake"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/format/flv/flvio"
	"github.com/nareix/joy4/utils/bits/pio"
)

var MaxProbePacketCount = 20

func NewMetadataByStreams(streams []av.CodecData) (metadata flvio.AMFMap, err error) {
	metadata = flvio.AMFMap{}

	for _, _stream := range streams {
		typ := _stream.Type()
		switch {
		case typ.IsVideo():
			stream := _stream.(av.VideoCodecData)
			switch typ {
			case av.H264:
				metadata["videocodecid"] = flvio.VIDEO_H264

			default:
				err = fmt.Errorf("flv: metadata: unsupported video codecType=%v", stream.Type())
				return
			}

			metadata["width"] = stream.Width()
			metadata["height"] = stream.Height()
			metadata["displayWidth"] = stream.Width()
			metadata["displayHeight"] = stream.Height()

		case typ.IsAudio():
			stream := _stream.(av.AudioCodecData)
			switch typ {
			case av.AAC:
				metadata["audiocodecid"] = flvio.SOUND_AAC

			case av.SPEEX:
				metadata["audiocodecid"] = flvio.SOUND_SPEEX

			default:
				err = fmt.Errorf("flv: metadata: unsupported audio codecType=%v", stream.Type())
				return
			}

			metadata["audiosamplerate"] = stream.SampleRate()
		}
	}

	return
}

type Prober struct {
	HasAudio, HasVideo             bool
	GotAudio, GotVideo             bool
	VideoStreamIdx, AudioStreamIdx int
	PushedCount                    int
	Streams                        []av.CodecData
	CachedPkts                     []av.Packet
}

func (self *Prober) CacheTag(_tag flvio.Tag, timestamp int32) {
	pkt, _ := self.TagToPacket(_tag, timestamp)
	self.CachedPkts = append(self.CachedPkts, pkt)
}

func (self *Prober) PushTag(tag flvio.Tag, timestamp int32) (err error) {
	self.PushedCount++

	if self.PushedCount > MaxProbePacketCount {
		err = fmt.Errorf("flv: max probe packet count reached")
		return
	}

	switch tag.Type {
	case flvio.TAG_VIDEO:
		switch tag.AVCPacketType {
		case flvio.AVC_SEQHDR:
			if !self.GotVideo {
				var stream h264parser.CodecData
				if stream, err = h264parser.NewCodecDataFromAVCDecoderConfRecord(tag.Data); err != nil {
					err = fmt.Errorf("flv: h264 seqhdr invalid")
					return
				}
				self.VideoStreamIdx = len(self.Streams)
				self.Streams = append(self.Streams, stream)
				self.GotVideo = true
			}

		case flvio.AVC_NALU:
			self.CacheTag(tag, timestamp)
		}

	case flvio.TAG_AUDIO:
		switch tag.SoundFormat {
		case flvio.SOUND_AAC:
			switch tag.AACPacketType {
			case flvio.AAC_SEQHDR:
				if !self.GotAudio {
					var stream aacparser.CodecData
					if stream, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(tag.Data); err != nil {
						err = fmt.Errorf("flv: aac seqhdr invalid")
						return
					}
					self.AudioStreamIdx = len(self.Streams)
					self.Streams = append(self.Streams, stream)
					self.GotAudio = true
				}

			case flvio.AAC_RAW:
				self.CacheTag(tag, timestamp)
			}

		case flvio.SOUND_SPEEX:
			if !self.GotAudio {
				stream := codec.NewSpeexCodecData(16000, tag.ChannelLayout())
				self.AudioStreamIdx = len(self.Streams)
				self.Streams = append(self.Streams, stream)
				self.GotAudio = true
				self.CacheTag(tag, timestamp)
			}

		case flvio.SOUND_NELLYMOSER:
			if !self.GotAudio {
				stream := fake.CodecData{
					CodecType_:     av.NELLYMOSER,
					SampleRate_:    16000,
					SampleFormat_:  av.S16,
					ChannelLayout_: tag.ChannelLayout(),
				}
				self.AudioStreamIdx = len(self.Streams)
				self.Streams = append(self.Streams, stream)
				self.GotAudio = true
				self.CacheTag(tag, timestamp)
			}

		}
	}

	return
}

func (self *Prober) Probed() (ok bool) {
	if self.HasAudio || self.HasVideo {
		if self.HasAudio == self.GotAudio && self.HasVideo == self.GotVideo {
			return true
		}
	} else {
		if self.PushedCount == MaxProbePacketCount {
			return true
		}
	}
	return
}

func (self *Prober) TagToPacket(tag flvio.Tag, timestamp int32) (pkt av.Packet, ok bool) {
	switch tag.Type {
	case flvio.TAG_VIDEO:
		pkt.Idx = int8(self.VideoStreamIdx)
		switch tag.AVCPacketType {
		case flvio.AVC_NALU:
			ok = true
			pkt.Data = tag.Data
			pkt.CompositionTime = flvio.TsToTime(tag.CompositionTime)
			pkt.IsKeyFrame = tag.FrameType == flvio.FRAME_KEY
		}

	case flvio.TAG_AUDIO:
		pkt.Idx = int8(self.AudioStreamIdx)
		switch tag.SoundFormat {
		case flvio.SOUND_AAC:
			switch tag.AACPacketType {
			case flvio.AAC_RAW:
				ok = true
				pkt.Data = tag.Data
			}

		case flvio.SOUND_SPEEX:
			ok = true
			pkt.Data = tag.Data

		case flvio.SOUND_NELLYMOSER:
			ok = true
			pkt.Data = tag.Data
		}
	}

	pkt.Time = flvio.TsToTime(timestamp)
	return
}

func (self *Prober) Empty() bool {
	return len(self.CachedPkts) == 0
}

func (self *Prober) PopPacket() av.Packet {
	pkt := self.CachedPkts[0]
	self.CachedPkts = self.CachedPkts[1:]
	return pkt
}

func CodecDataToTag(stream av.CodecData) (_tag flvio.Tag, ok bool, err error) {
	switch stream.Type() {
	case av.H264:
		h264 := stream.(h264parser.CodecData)
		tag := flvio.Tag{
			Type:          flvio.TAG_VIDEO,
			AVCPacketType: flvio.AVC_SEQHDR,
			CodecID:       flvio.VIDEO_H264,
			Data:          h264.AVCDecoderConfRecordBytes(),
			FrameType:     flvio.FRAME_KEY,
		}
		ok = true
		_tag = tag

	case av.NELLYMOSER:
	case av.SPEEX:

	case av.AAC:
		aac := stream.(aacparser.CodecData)
		tag := flvio.Tag{
			Type:          flvio.TAG_AUDIO,
			SoundFormat:   flvio.SOUND_AAC,
			SoundRate:     flvio.SOUND_44Khz,
			AACPacketType: flvio.AAC_SEQHDR,
			Data:          aac.MPEG4AudioConfigBytes(),
		}
		switch aac.SampleFormat().BytesPerSample() {
		case 1:
			tag.SoundSize = flvio.SOUND_8BIT
		default:
			tag.SoundSize = flvio.SOUND_16BIT
		}
		switch aac.ChannelLayout().Count() {
		case 1:
			tag.SoundType = flvio.SOUND_MONO
		case 2:
			tag.SoundType = flvio.SOUND_STEREO
		}
		ok = true
		_tag = tag

	default:
		err = fmt.Errorf("flv: unspported codecType=%v", stream.Type())
		return
	}
	return
}

func PacketToTag(pkt av.Packet, stream av.CodecData) (tag flvio.Tag, timestamp int32) {
	switch stream.Type() {
	case av.H264:
		tag = flvio.Tag{
			Type:            flvio.TAG_VIDEO,
			AVCPacketType:   flvio.AVC_NALU,
			CodecID:         flvio.VIDEO_H264,
			Data:            pkt.Data,
			CompositionTime: flvio.TimeToTs(pkt.CompositionTime),
		}
		if pkt.IsKeyFrame {
			tag.FrameType = flvio.FRAME_KEY
		} else {
			tag.FrameType = flvio.FRAME_INTER
		}

	case av.AAC:
		tag = flvio.Tag{
			Type:          flvio.TAG_AUDIO,
			SoundFormat:   flvio.SOUND_AAC,
			SoundRate:     flvio.SOUND_44Khz,
			AACPacketType: flvio.AAC_RAW,
			Data:          pkt.Data,
		}
		astream := stream.(av.AudioCodecData)
		switch astream.SampleFormat().BytesPerSample() {
		case 1:
			tag.SoundSize = flvio.SOUND_8BIT
		default:
			tag.SoundSize = flvio.SOUND_16BIT
		}
		switch astream.ChannelLayout().Count() {
		case 1:
			tag.SoundType = flvio.SOUND_MONO
		case 2:
			tag.SoundType = flvio.SOUND_STEREO
		}

	case av.SPEEX:
		tag = flvio.Tag{
			Type:        flvio.TAG_AUDIO,
			SoundFormat: flvio.SOUND_SPEEX,
			Data:        pkt.Data,
		}

	case av.NELLYMOSER:
		tag = flvio.Tag{
			Type:        flvio.TAG_AUDIO,
			SoundFormat: flvio.SOUND_NELLYMOSER,
			Data:        pkt.Data,
		}
	}

	timestamp = flvio.TimeToTs(pkt.Time)
	return
}

type Muxer struct {
	bufw    writeFlusher
	b       []byte
	streams []av.CodecData
}

type writeFlusher interface {
	io.Writer
	Flush() error
}

func NewMuxerWriteFlusher(w writeFlusher) *Muxer {
	return &Muxer{
		bufw: w,
		b:    make([]byte, 256),
	}
}

func NewMuxer(w io.Writer) *Muxer {
	return NewMuxerWriteFlusher(bufio.NewWriterSize(w, pio.RecommendBufioSize))
}

var CodecTypes = []av.CodecType{av.H264, av.AAC, av.SPEEX}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	var flags uint8
	for _, stream := range streams {
		if stream.Type().IsVideo() {
			flags |= flvio.FILE_HAS_VIDEO
		} else if stream.Type().IsAudio() {
			flags |= flvio.FILE_HAS_AUDIO
		}
	}

	n := flvio.FillFileHeader(self.b, flags)
	if _, err = self.bufw.Write(self.b[:n]); err != nil {
		return
	}

	for _, stream := range streams {
		var tag flvio.Tag
		var ok bool
		if tag, ok, err = CodecDataToTag(stream); err != nil {
			return
		}
		if ok {
			if err = flvio.WriteTag(self.bufw, tag, 0, self.b); err != nil {
				return
			}
		}
	}

	self.streams = streams
	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	stream := self.streams[pkt.Idx]
	tag, timestamp := PacketToTag(pkt, stream)

	if err = flvio.WriteTag(self.bufw, tag, timestamp, self.b); err != nil {
		return
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {
	if err = self.bufw.Flush(); err != nil {
		return
	}
	return
}

type Demuxer struct {
	prober *Prober
	bufr   *bufio.Reader
	b      []byte
	stage  int
}

func NewDemuxer(r io.Reader) *Demuxer {
	return &Demuxer{
		bufr:   bufio.NewReaderSize(r, pio.RecommendBufioSize),
		prober: &Prober{},
		b:      make([]byte, 256),
	}
}

func (self *Demuxer) prepare() (err error) {
	for self.stage < 2 {
		switch self.stage {
		case 0:
			if _, err = io.ReadFull(self.bufr, self.b[:flvio.FileHeaderLength]); err != nil {
				return
			}
			var flags uint8
			var skip int
			if flags, skip, err = flvio.ParseFileHeader(self.b); err != nil {
				return
			}
			if _, err = self.bufr.Discard(skip); err != nil {
				return
			}
			if flags&flvio.FILE_HAS_AUDIO != 0 {
				self.prober.HasAudio = true
			}
			if flags&flvio.FILE_HAS_VIDEO != 0 {
				self.prober.HasVideo = true
			}
			self.stage++

		case 1:
			for !self.prober.Probed() {
				var tag flvio.Tag
				var timestamp int32
				if tag, timestamp, err = flvio.ReadTag(self.bufr, self.b); err != nil {
					return
				}
				if err = self.prober.PushTag(tag, timestamp); err != nil {
					return
				}
			}
			self.stage++
		}
	}
	return
}

func (self *Demuxer) Streams() (streams []av.CodecData, err error) {
	if err = self.prepare(); err != nil {
		return
	}
	streams = self.prober.Streams
	return
}

func (self *Demuxer) ReadPacket() (pkt av.Packet, err error) {
	if err = self.prepare(); err != nil {
		return
	}

	if !self.prober.Empty() {
		pkt = self.prober.PopPacket()
		return
	}

	for {
		var tag flvio.Tag
		var timestamp int32
		if tag, timestamp, err = flvio.ReadTag(self.bufr, self.b); err != nil {
			return
		}

		var ok bool
		if pkt, ok = self.prober.TagToPacket(tag, timestamp); ok {
			return
		}
	}

	return
}

func Handler(h *avutil.RegisterHandler) {
	h.Probe = func(b []byte) bool {
		return b[0] == 'F' && b[1] == 'L' && b[2] == 'V'
	}

	h.Ext = ".flv"

	h.ReaderDemuxer = func(r io.Reader) av.Demuxer {
		return NewDemuxer(r)
	}

	h.WriterMuxer = func(w io.Writer) av.Muxer {
		return NewMuxer(w)
	}

	h.CodecTypes = CodecTypes
}
