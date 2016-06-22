package transcode

import (
	"fmt"
	"time"
	"github.com/nareix/av"
	"github.com/nareix/av/pktque"
)

const debug = true

type tStream struct {
	codec av.CodecData
	timeline *pktque.Timeline
	aencodec, adecodec av.AudioCodecData
	aenc av.AudioEncoder
	adec av.AudioDecoder
}

type Transcoder struct {
	FindAudioDecoderEncoder func(codec av.AudioCodecData) (ok bool, err error, dec av.AudioDecoder, enc av.AudioEncoder)
	streams                 []*tStream
}

func (self *Transcoder) Setup(streams []av.CodecData) (err error) {
	self.streams = []*tStream{}

	for _, stream := range streams {
		ts := &tStream{codec: stream}
		if stream.Type().IsAudio() {
			if self.FindAudioDecoderEncoder != nil {
				var ok bool
				var enc av.AudioEncoder
				var dec av.AudioDecoder
				ok, err, dec, enc = self.FindAudioDecoderEncoder(stream.(av.AudioCodecData))
				if ok {
					if err != nil {
						return
					}
					ts.timeline = &pktque.Timeline{}
					ts.codec = enc.CodecData()
					ts.aencodec = ts.codec.(av.AudioCodecData)
					ts.adecodec = stream.(av.AudioCodecData)
					ts.aenc = enc
					ts.adec = dec
				}
			}
		}
		self.streams = append(self.streams, ts)
	}
	return
}

func (self *tStream) audioDecodeAndEncode(inpkt av.Packet) (outpkts []av.Packet, err error) {
	var dur time.Duration
	var frame av.AudioFrame
	var ok bool
	if ok, frame, err = self.adec.Decode(inpkt.Data); err != nil {
		return
	}
	if !ok {
		return
	}

	if dur, err = self.adecodec.PacketDuration(inpkt.Data); err != nil {
		err = fmt.Errorf("transcode: PacketDuration() failed for input stream #%d", inpkt.Idx)
		return
	}

	if debug {
		fmt.Println("transcode: push", inpkt.Time, dur)
	}
	self.timeline.Push(inpkt.Time, dur)

	var _outpkts [][]byte
	if _outpkts, err = self.aenc.Encode(frame); err != nil {
		return
	}
	for _, _outpkt := range _outpkts {
		if dur, err = self.aencodec.PacketDuration(_outpkt); err != nil {
			err = fmt.Errorf("transcode: PacketDuration() failed for output stream #%d", inpkt.Idx)
			return
		}
		outpkt := av.Packet{Idx: inpkt.Idx, Data: _outpkt}
		outpkt.Time = self.timeline.Pop(dur)

		if debug {
			fmt.Println("transcode: pop", outpkt.Time)
		}

		outpkts = append(outpkts, outpkt)
	}

	return
}

func (self *Transcoder) Do(pkt av.Packet) (out []av.Packet, err error) {
	stream := self.streams[pkt.Idx]
	if stream.aenc != nil && stream.adec != nil {
		if out, err = stream.audioDecodeAndEncode(pkt); err != nil {
			return
		}
	} else {
		out = append(out, pkt)
	}
	return
}

func (self *Transcoder) Streams() (streams []av.CodecData, err error) {
	for _, stream := range self.streams {
		streams = append(streams, stream.codec)
	}
	return
}

func (self *Transcoder) Close() {
	for _, stream := range self.streams {
		if stream.aenc != nil {
			stream.aenc.Close()
			stream.aenc = nil
		}
		if stream.adec != nil {
			stream.adec.Close()
			stream.adec = nil
		}
	}
	self.streams = []*tStream{}
}

type Muxer struct {
	Muxer      av.Muxer
	Transcoder *Transcoder
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	if err = self.Transcoder.Setup(streams); err != nil {
		return
	}
	var newstreams []av.CodecData
	if newstreams, err = self.Transcoder.Streams(); err != nil {
		return
	}
	if err = self.Muxer.WriteHeader(newstreams); err != nil {
		return
	}
	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	var outpkts []av.Packet
	if outpkts, err = self.Transcoder.Do(pkt); err != nil {
		return
	}
	for _, pkt := range outpkts {
		if err = self.Muxer.WritePacket(pkt); err != nil {
			return
		}
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {
	// TODO: do flush
	if err = self.Muxer.WriteTrailer(); err != nil {
		return
	}
	return
}

type Demuxer struct {
	Demuxer    av.Demuxer
	Transcoder *Transcoder
	outpkts []av.Packet
}

func (self *Demuxer) Setup() (err error) {
	var streams []av.CodecData
	if streams, err = self.Demuxer.Streams(); err != nil {
		return
	}
	if err = self.Transcoder.Setup(streams); err != nil {
		return
	}
	return
}

func (self *Demuxer) ReadPacket() (pkt av.Packet, err error) {
	for {
		if len(self.outpkts) > 0 {
			pkt = self.outpkts[0]
			self.outpkts = self.outpkts[1:]
			return
		}
		var rpkt av.Packet
		if rpkt, err = self.Demuxer.ReadPacket(); err != nil {
			return
		}
		if self.outpkts, err = self.Transcoder.Do(rpkt); err != nil {
			return
		}
	}
	return
}

func (self *Demuxer) Streams() ([]av.CodecData, error) {
	return self.Transcoder.Streams()
}

func (self *Demuxer) Close() {
	self.Transcoder.Close()
}
