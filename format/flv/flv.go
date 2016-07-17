package flv

import (
	"fmt"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/codec"
	"github.com/nareix/joy4/codec/fake"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/pio"
	"github.com/nareix/joy4/format/flv/flvio"
	"io"
	"bufio"
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
	HasAudio, HasVideo bool
	GotAudio, GotVideo bool
	VideoStreamIdx, AudioStreamIdx int
	PushedCount int
	Streams []av.CodecData
	CachedPkts []av.Packet
}

func (self *Prober) CacheTag(_tag flvio.Tag, timestamp int32) {
	pkt, _ := self.TagToPacket(_tag, timestamp)
	self.CachedPkts = append(self.CachedPkts, pkt)
}

func (self *Prober) PushTag(_tag flvio.Tag, timestamp int32) (err error) {
	self.PushedCount++

	if self.PushedCount > MaxProbePacketCount {
		err = fmt.Errorf("flv: max probe packet count reached")
		return
	}

	switch tag := _tag.(type) {
	case *flvio.Videodata:
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

	case *flvio.Audiodata:
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
					CodecType_: av.NELLYMOSER,
					SampleRate_: 16000,
					SampleFormat_: av.S16,
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

func (self *Prober) TagToPacket(_tag flvio.Tag, timestamp int32) (pkt av.Packet, ok bool) {
	switch tag := _tag.(type) {
	case *flvio.Videodata:
		pkt.Idx = int8(self.VideoStreamIdx)
		switch tag.AVCPacketType {
		case flvio.AVC_NALU:
			ok = true
			pkt.Data = tag.Data
			pkt.CompositionTime = flvio.TsToTime(tag.CompositionTime)
			pkt.IsKeyFrame = tag.FrameType == flvio.FRAME_KEY
		}

	case *flvio.Audiodata:
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
		tag := &flvio.Videodata{
			AVCPacketType: flvio.AVC_SEQHDR,
			CodecID: flvio.VIDEO_H264,
			Data: h264.AVCDecoderConfRecordBytes(),
			FrameType: flvio.FRAME_KEY,
		}
		ok = true
		_tag = tag

	case av.NELLYMOSER:
	case av.SPEEX:

	case av.AAC:
		aac := stream.(aacparser.CodecData)
		tag := &flvio.Audiodata{
			SoundFormat: flvio.SOUND_AAC,
			SoundRate: flvio.SOUND_44Khz,
			AACPacketType: flvio.AAC_SEQHDR,
			Data: aac.MPEG4AudioConfigBytes(),
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

func PacketToTag(pkt av.Packet, stream av.CodecData) (_tag flvio.Tag, timestamp int32) {
	switch stream.Type() {
	case av.H264:
		tag := &flvio.Videodata{
			AVCPacketType: flvio.AVC_NALU,
			CodecID: flvio.VIDEO_H264,
			Data: pkt.Data,
			CompositionTime: flvio.TimeToTs(pkt.CompositionTime),
		}
		if pkt.IsKeyFrame {
			tag.FrameType = flvio.FRAME_KEY
		} else {
			tag.FrameType = flvio.FRAME_INTER
		}
		_tag = tag

	case av.AAC:
		tag := &flvio.Audiodata{
			SoundFormat: flvio.SOUND_AAC,
			SoundRate: flvio.SOUND_44Khz,
			AACPacketType: flvio.AAC_RAW,
			Data: pkt.Data,
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
		_tag = tag

	case av.SPEEX:
		tag := &flvio.Audiodata{
			SoundFormat: flvio.SOUND_SPEEX,
			Data: pkt.Data,
		}
		_tag = tag

	case av.NELLYMOSER:
		tag := &flvio.Audiodata{
			SoundFormat: flvio.SOUND_NELLYMOSER,
			Data: pkt.Data,
		}
		_tag = tag
	}

	timestamp = flvio.TimeToTs(pkt.Time)
	return
}

type Muxer struct {
	pw *pio.Writer
	bw *bufio.Writer
	streams []av.CodecData
}

func NewMuxer(w io.Writer) *Muxer {
	self := &Muxer{}
	self.bw = bufio.NewWriterSize(w, pio.RecommendBufioSize)
	self.pw = pio.NewWriter(self.bw)
	return self
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

	if err = flvio.WriteFileHeader(self.pw, flags); err != nil {
		return
	}

	for _, stream := range streams {
		var tag flvio.Tag
		var ok bool
		if tag, ok, err = CodecDataToTag(stream); err != nil {
			return
		}
		if ok {
			if err = flvio.WriteTag(self.pw, tag, 0); err != nil {
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

	if err = flvio.WriteTag(self.pw, tag, timestamp); err != nil {
		return
	}

	return
}

func (self *Muxer) WriteTrailer() (err error) {
	if err = self.bw.Flush(); err != nil {
		return
	}
	return
}

type Demuxer struct {
	prober *Prober
	pr *pio.Reader
	stage int
}

func NewDemuxer(r io.Reader) *Demuxer {
	return &Demuxer{
		pr: pio.NewReader(bufio.NewReaderSize(r, pio.RecommendBufioSize)),
		prober: &Prober{},
	}
}

func (self *Demuxer) prepare() (err error) {
	for self.stage < 2 {
		switch self.stage {
		case 0:
			var flags uint8
			if flags, err = flvio.ReadFileHeader(self.pr); err != nil {
				return
			}
			if flags & flvio.FILE_HAS_AUDIO != 0 {
				self.prober.HasAudio = true
			}
			if flags & flvio.FILE_HAS_VIDEO != 0 {
				self.prober.HasVideo = true
			}
			self.stage++

		case 1:
			for !self.prober.Probed() {
				var tag flvio.Tag
				var timestamp int32
				if tag, timestamp, err = flvio.ReadTag(self.pr); err != nil {
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
		if tag, timestamp, err = flvio.ReadTag(self.pr); err != nil {
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

