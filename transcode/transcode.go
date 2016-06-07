package transcode

import (
	"github.com/nareix/av"
	"github.com/nareix/av/pktreorder"
	"fmt"
)

const debug = false

type tstream struct {
	av.CodecData
	aenc av.AudioEncoder
	adec av.AudioDecoder
}

type Transcoder struct {
	FindAudioDecoderEncoder func(codec av.AudioCodecData) (ok bool, err error, dec av.AudioDecoder, enc av.AudioEncoder)
	streams []*tstream
	queue *pktreorder.Queue
}

func (self *Transcoder) Setup(streams []av.CodecData) (err error) {
	self.streams = []*tstream{}

	for _, stream := range streams {
		ts := &tstream{CodecData: stream}
		if stream.IsAudio() {
			if self.FindAudioDecoderEncoder != nil {
				var ok bool
				var enc av.AudioEncoder
				var dec av.AudioDecoder
				ok, err, dec, enc = self.FindAudioDecoderEncoder(stream.(av.AudioCodecData))
				if ok {
					if err != nil {
						return
					}
					ts.CodecData = enc.CodecData()
					ts.aenc = enc
					ts.adec = dec
				}
			}
		}
		self.streams = append(self.streams, ts)
	}

	self.queue = &pktreorder.Queue{}
	self.queue.Alloc(self.Streams())
	return
}

func (self *Transcoder) decodeAndEncode(stream *tstream, i int, pkt av.Packet) (err error) {
	var frame av.AudioFrame
	var ok bool
	if ok, frame, err = stream.adec.Decode(pkt.Data); err != nil {
		return
	}
	if ok {
		var pkts []av.Packet
		if pkts, err = stream.aenc.Encode(frame); err != nil {
			return
		}
		for _, pkt := range pkts {
			self.queue.WritePacket(i, pkt)
		}
	}
	return
}

func (self *Transcoder) WritePacket(i int, pkt av.Packet) {
	if debug {
		fmt.Println("transcode: Transcoder.WritePacket", i, len(pkt.Data), fmt.Sprintf("%.2f", pkt.Duration))
		fmt.Println("transcode: Transcoder.CanReadPacket", self.CanReadPacket())
	}

	stream := self.streams[i]
	if stream.aenc != nil && stream.adec != nil {
		if err := self.decodeAndEncode(stream, i, pkt); err != nil {
			self.queue.EndWritePacket(err)
		}
	} else {
		self.queue.WritePacket(i, pkt)
	}

	return
}

func (self *Transcoder) EndWritePacket(err error) {
	self.queue.EndWritePacket(err)
}

func (self *Transcoder) CanReadPacket() bool {
	return self.queue.CanReadPacket()
}

func (self *Transcoder) CanWritePacket() bool {
	return self.queue.CanWritePacket()
}

func (self *Transcoder) Error() error {
	return self.queue.Error()
}

func (self *Transcoder) ReadPacket() (i int, pkt av.Packet, err error) {
	return self.queue.ReadPacket()
}

func (self *Transcoder) Streams() (streams []av.CodecData) {
	for _, stream := range self.streams {
		streams = append(streams, stream.CodecData)
	}
	return
}

func (self *Transcoder) Close() {
	for _, stream := range self.streams {
		if stream.aenc != nil {
			stream.aenc.Close()
		}
		if stream.adec != nil {
			stream.adec.Close()
		}
	}
	self.streams = []*tstream{}
}

type Muxer struct {
	Muxer av.Muxer
	Transcoder *Transcoder
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	if err = self.Transcoder.Setup(streams); err != nil {
		return
	}
	if err = self.Muxer.WriteHeader(self.Transcoder.Streams()); err != nil {
		return
	}
	return
}

func (self *Muxer) WritePacket(i int, pkt av.Packet) (err error) {
	self.Transcoder.WritePacket(i, pkt)
	if self.Transcoder.CanReadPacket() {
		if i, pkt, rerr := self.Transcoder.ReadPacket(); rerr != nil {
			err = rerr
			return
		} else {
			if werr := self.Muxer.WritePacket(i, pkt); werr != nil {
				self.Transcoder.EndWritePacket(werr)
			}
		}
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {
	self.Transcoder.EndWritePacket(nil)
	for {
		if i, pkt, rerr := self.Transcoder.ReadPacket(); rerr != nil {
			break
		} else {
			if werr := self.Muxer.WritePacket(i, pkt); werr != nil {
				err = werr
				return
			}
		}
	}
	if err = self.Muxer.WriteTrailer(); err != nil {
		return
	}
	return
}

type Demuxer struct {
	Demuxer av.Demuxer
	Transcoder *Transcoder
}

func (self *Demuxer) Setup() (err error) {
	if err = self.Transcoder.Setup(self.Demuxer.Streams()); err != nil {
		return
	}
	return
}

func (self *Demuxer) ReadPacket() (i int, pkt av.Packet, err error) {
	for {
		if self.Transcoder.CanReadPacket() {
			return self.Transcoder.ReadPacket()
		} else if self.Transcoder.CanWritePacket() {
			if i, pkt, err := self.Demuxer.ReadPacket(); err != nil {
				self.Transcoder.EndWritePacket(err)
			} else {
				self.Transcoder.WritePacket(i, pkt)
			}
		} else {
			err = self.Transcoder.Error()
			return
		}
	}
}

func (self *Demuxer) Streams() []av.CodecData {
	return self.Transcoder.Streams()
}

func (self *Demuxer) Close() {
	self.Transcoder.Close()
}

