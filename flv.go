package flv

import (
	"time"
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/av/avutil"
	"github.com/nareix/codec/h264parser"
	"github.com/nareix/codec/aacparser"
	"github.com/nareix/pio"
	"github.com/nareix/flv/flvio"
	"io"
	"bufio"
)

type Muxer struct {
	pw *pio.Writer
	streams []av.CodecData
}

func NewMuxer(w io.Writer) *Muxer {
	self := &Muxer{}
	self.pw = pio.NewWriter(w)
	return self
}

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
		var _tag flvio.Tag

		switch stream.Type() {
		case av.H264:
			h264 := stream.(h264parser.CodecData)
			tag := &flvio.Videodata{
				AVCPacketType: flvio.AVC_SEQHDR,
				CodecID: flvio.VIDEO_H264,
				Data: h264.AVCDecoderConfRecordBytes(),
			}
			_tag = tag

		case av.AAC:
			aac := stream.(aacparser.CodecData)
			tag := flvio.MakeAACAudiodata(aac, aac.MPEG4AudioConfigBytes())
			tag.AACPacketType = flvio.AAC_SEQHDR
			_tag = &tag

		default:
			err = fmt.Errorf("flv: unspported codecType=%v", stream.Type())
			return
		}

		if err = flvio.WriteTag(self.pw, _tag, 0); err != nil {
			return
		}
	}

	self.streams = streams
	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	stream := self.streams[pkt.Idx]
	var _tag flvio.Tag

	switch stream.Type() {
	case av.H264:
		if typ := h264parser.CheckNALUsType(pkt.Data); typ != h264parser.NALU_RAW {
			err = fmt.Errorf("flv: h264 nalu format=%d invalid", typ)
			return
		}
		var b [4]byte
		pio.PutU32BE(b[:], uint32(len(pkt.Data)))
		tag := &flvio.Videodata{
			AVCPacketType: flvio.AVC_NALU,
			CodecID: flvio.VIDEO_H264,
			Datav: [][]byte{b[:], pkt.Data},
			CompositionTime: timeToTs(pkt.CompositionTime),
		}
		if pkt.IsKeyFrame {
			tag.FrameType = flvio.FRAME_KEY
		} else {
			tag.FrameType = flvio.FRAME_INTER
		}
		_tag = tag

	case av.AAC:
		tag := flvio.MakeAACAudiodata(stream.(av.AudioCodecData), pkt.Data)
		_tag = &tag
	}

	if err = flvio.WriteTag(self.pw, _tag, timeToTs(pkt.Time)); err != nil {
		return
	}

	return
}

func (self *Muxer) WriteTrailer() (err error) {
	return
}

type flvStream struct {
	av.CodecData
	lastts int32
	tm time.Duration
}

type Demuxer struct {
	streams []*flvStream
	videostreamidx int
	audiostreamidx int
	pr *pio.Reader
	probepkts []flvio.Tag
	probed bool
}

func NewDemuxer(r io.Reader) *Demuxer {
	return &Demuxer{
		pr: pio.NewReader(bufio.NewReaderSize(r, 128)),
	}
}

func (self *Demuxer) probe() (err error) {
	if self.probed {
		return
	}

	var flags, got uint8
	if flags, err = flvio.ReadFileHeader(self.pr); err != nil {
		return
	}
	flags &= flvio.FILE_HAS_AUDIO|flvio.FILE_HAS_VIDEO

	for {
		var _tag flvio.Tag
		if _tag, _, err = flvio.ReadTag(self.pr); err != nil {
			return
		}

		switch tag := _tag.(type) {
		case *flvio.Videodata:
			switch tag.CodecID {
			case flvio.VIDEO_H264:
				switch tag.AVCPacketType {
				case flvio.AVC_SEQHDR:
					var codec h264parser.CodecData
					if codec, err = h264parser.NewCodecDataFromAVCDecoderConfRecord(tag.Data); err != nil {
						err = fmt.Errorf("flv: h264 seqhdr invalid")
						return
					}
					self.videostreamidx = len(self.streams)
					self.streams = append(self.streams, &flvStream{CodecData: codec})
					got |= flvio.FILE_HAS_VIDEO

				case flvio.AVC_NALU:
					self.probepkts = append(self.probepkts, tag)
				}

			default:
				err = fmt.Errorf("flv: unspported video CodecID=%d", tag.CodecID)
				return
			}

		case *flvio.Audiodata:
			switch tag.SoundFormat {
			case flvio.SOUND_AAC:
				switch tag.AACPacketType {
				case flvio.AAC_SEQHDR:
					var codec aacparser.CodecData
					if codec, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(tag.Data); err != nil {
						err = fmt.Errorf("flv: aac seqhdr invalid")
						return
					}
					self.audiostreamidx = len(self.streams)
					self.streams = append(self.streams, &flvStream{CodecData: codec})
					got |= flvio.FILE_HAS_AUDIO

				case flvio.AAC_RAW:
					self.probepkts = append(self.probepkts, tag)
				}

			default:
				err = fmt.Errorf("flv: unspported audio SoundFormat=%d", tag.SoundFormat)
				return
			}
		}

		if got == flags {
			break
		}
	}

	self.probed = true
	return
}

func (self *Demuxer) Streams() (streams []av.CodecData, err error) {
	if err = self.probe(); err != nil {
		return
	}
	for _, stream := range self.streams {
		streams = append(streams, stream.CodecData)
	}
	return
}

func tsToTime(ts int32) time.Duration {
	return time.Millisecond*time.Duration(ts)
}

func timeToTs(tm time.Duration) int32 {
	return int32(tm / time.Millisecond)
}

func (self *Demuxer) ReadPacket() (pkt av.Packet, err error) {
	if err = self.probe(); err != nil {
		return
	}

	var timestamp int32
	var stream *flvStream

	loop: for {
		var _tag flvio.Tag
		if len(self.probepkts) > 0 {
			_tag = self.probepkts[0]
			self.probepkts = self.probepkts[1:]
		} else {
			if _tag, timestamp, err = flvio.ReadTag(self.pr); err != nil {
				return
			}
		}

		switch tag := _tag.(type) {
		case *flvio.Videodata:
			if tag.AVCPacketType == flvio.AVC_NALU {
				stream = self.streams[self.videostreamidx]
				pkt.Idx = int8(self.videostreamidx)
				pkt.CompositionTime = tsToTime(tag.CompositionTime)
				pkt.IsKeyFrame = tag.FrameType == flvio.FRAME_KEY
				if typ := h264parser.CheckNALUsType(tag.Data); typ != h264parser.NALU_AVCC {
					err = fmt.Errorf("flv: h264 nalu format=%d invalid", typ)
					return
				}
				pkt.Data = tag.Data[4:]
				break loop
			}

		case *flvio.Audiodata:
			if tag.AACPacketType == flvio.AAC_RAW {
				stream = self.streams[self.audiostreamidx]
				pkt.Idx = int8(self.audiostreamidx)
				pkt.Data = tag.Data
				break loop
			}
		}
	}

	if stream.lastts == 0 {
		stream.tm = tsToTime(timestamp)
	} else {
		diff := timestamp - stream.lastts
		stream.tm += tsToTime(diff)
	}
	stream.lastts = timestamp
	pkt.Time = stream.tm

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
}

