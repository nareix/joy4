
// Package transcoder implements Transcoder based on Muxer/Demuxer and AudioEncoder/AudioDecoder interface.
package transcode

import (
	"fmt"
	"time"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/cgo/ffmpeg"
)

var Debug bool

type tStream struct {
	codec av.CodecData
	timeline *pktque.Timeline
	aencodec, adecodec av.AudioCodecData
	aenc av.AudioEncoder
	adec av.AudioDecoder
	vencodec, vdecodec av.VideoCodecData
	venc *ffmpeg.VideoEncoder
	vdec *ffmpeg.VideoDecoder
}

type Options struct {
	// check if transcode is needed, and create the AudioDecoder and AudioEncoder.
	FindAudioDecoderEncoder func(codec av.AudioCodecData, i int) (
		need bool, dec av.AudioDecoder, enc av.AudioEncoder, err error,
	)
	FindVideoDecoderEncoder func(codec av.VideoCodecData, i int) (
		need bool, dec *ffmpeg.VideoDecoder, enc *ffmpeg.VideoEncoder, err error,
	)
}

type Transcoder struct {
	streams                 []*tStream
}

func NewTranscoder(streams []av.CodecData, options Options) (_self *Transcoder, err error) {
	self := &Transcoder{}
	self.streams = []*tStream{}

	for i, stream := range streams {
		ts := &tStream{codec: stream}
		if stream.Type().IsAudio() {
			if options.FindAudioDecoderEncoder != nil {
				var ok bool
				var enc av.AudioEncoder
				var dec av.AudioDecoder
				ok, dec, enc, err = options.FindAudioDecoderEncoder(stream.(av.AudioCodecData), i)
				if ok {
					if err != nil {
						return
					}
					ts.timeline = &pktque.Timeline{}
					if ts.codec, err = enc.CodecData(); err != nil {
						return
					}
					ts.aencodec = ts.codec.(av.AudioCodecData)
					ts.adecodec = stream.(av.AudioCodecData)
					ts.aenc = enc
					ts.adec = dec
				}
			}
		} else if stream.Type().IsVideo() {
			if options.FindVideoDecoderEncoder != nil {
				var ok bool
				var enc *ffmpeg.VideoEncoder
				var dec *ffmpeg.VideoDecoder
				ok, dec, enc, err = options.FindVideoDecoderEncoder(stream.(av.VideoCodecData), i)
				if ok {
					if err != nil {
						return
					}
					ts.timeline = &pktque.Timeline{}
					if ts.codec, err = enc.CodecData(); err != nil {
						return
					}
					ts.vencodec = ts.codec.(av.VideoCodecData)
					ts.vdecodec = stream.(av.VideoCodecData)
					ts.venc = enc
					ts.vdec = dec
				}
			}
		}
		self.streams = append(self.streams, ts)
	}

	_self = self
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

	if Debug {
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

		if Debug {
			fmt.Println("transcode: pop", outpkt.Time, dur)
		}

		outpkts = append(outpkts, outpkt)
	}

	return
}

func (self *tStream) videoDecodeAndEncode(inpkt av.Packet) (outpkts []av.Packet, err error) {
	var dur time.Duration
	var frame *ffmpeg.VideoFrame
	if frame, err = self.vdec.Decode(inpkt.Data); err != nil || frame == nil {
		return
	}

	if dur, err = self.vdecodec.PacketDuration(inpkt.Data); err != nil {
		err = fmt.Errorf("transcode: PacketDuration() failed for input stream #%d", inpkt.Idx)
		return
	}

	if Debug {
		fmt.Println("transcode: push", inpkt.Time, dur)
	}
	self.timeline.Push(inpkt.Time, dur)

	var _outpkts [][]byte
	if _outpkts, err = self.venc.Encode(frame); err != nil {
		return
	}
	for _, _outpkt := range _outpkts {
		if fpsNum, fpsDen := self.vencodec.Framerate(); fpsNum <= 0 || fpsDen <= 0 {
			// FIXME this is a bit hacky
			// Read codec data after encoding (because the sps and pps are not ready before the first keyframe is encoded)
			var codecData av.VideoCodecData
			codecData, err = self.venc.CodecData()
			if err != nil {
				return
			}
			self.vencodec = codecData.(av.VideoCodecData)
		}
		if dur, err = self.vencodec.PacketDuration(_outpkt); err != nil {
			err = fmt.Errorf("transcode: PacketDuration() failed for output stream #%d", inpkt.Idx)
			return
		}
		outpkt := av.Packet{Idx: inpkt.Idx, Data: _outpkt}
		outpkt.Time = self.timeline.Pop(dur)

		if Debug {
			fmt.Println("transcode: pop", outpkt.Time, dur)
		}

		outpkts = append(outpkts, outpkt)
	}
	return
}

// Do the transcode.
// 
// In audio transcoding one Packet may transcode into many Packets
// packet time will be adjusted automatically.
func (self *Transcoder) Do(pkt av.Packet) (out []av.Packet, err error) {
	stream := self.streams[pkt.Idx]
	if stream.aenc != nil && stream.adec != nil {
		if out, err = stream.audioDecodeAndEncode(pkt); err != nil {
			return
		}
	} else if stream.venc != nil && stream.vdec != nil {
		if out, err = stream.videoDecodeAndEncode(pkt); err != nil {
			return
		}
	} else {
		out = append(out, pkt)
	}
	return
}

// Get CodecDatas after transcoding.
func (self *Transcoder) Streams() (streams []av.CodecData, err error) {
	for _, stream := range self.streams {
		streams = append(streams, stream.codec)
	}
	return
}

// Close transcoder, close related encoder and decoders.
func (self *Transcoder) Close() (err error) {
	for _, stream := range self.streams {
		if stream.aenc != nil {
			stream.aenc.Close()
			stream.aenc = nil
		}
		if stream.adec != nil {
			stream.adec.Close()
			stream.adec = nil
		}
		if stream.venc != nil {
			stream.venc.Close()
			stream.venc = nil
		}
		if stream.vdec != nil {
			stream.vdec.Close()
			stream.vdec = nil
		}
	}
	self.streams = nil
	return
}

// Wrap transcoder and origin Muxer into new Muxer.
// Write to new Muxer will do transcoding automatically.
type Muxer struct {
	av.Muxer // origin Muxer
	Options // transcode options
	transcoder *Transcoder
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	if self.transcoder, err = NewTranscoder(streams, self.Options); err != nil {
		return
	}
	var newstreams []av.CodecData
	if newstreams, err = self.transcoder.Streams(); err != nil {
		return
	}
	if err = self.Muxer.WriteHeader(newstreams); err != nil {
		return
	}
	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	var outpkts []av.Packet
	if outpkts, err = self.transcoder.Do(pkt); err != nil {
		return
	}
	for _, pkt := range outpkts {
		if err = self.Muxer.WritePacket(pkt); err != nil {
			return
		}
	}
	return
}

func (self *Muxer) Close() (err error) {
	if self.transcoder != nil {
		return self.transcoder.Close()
	}
	return
}

// Wrap transcoder and origin Demuxer into new Demuxer.
// Read this Demuxer will do transcoding automatically.
type Demuxer struct {
	av.Demuxer
	Options
	transcoder *Transcoder
	outpkts []av.Packet
}

func (self *Demuxer) prepare() (err error) {
	if self.transcoder == nil {
		var streams []av.CodecData
		if streams, err = self.Demuxer.Streams(); err != nil {
			return
		}
		if self.transcoder, err = NewTranscoder(streams, self.Options); err != nil {
			return
		}
	}
	return
}

func (self *Demuxer) ReadPacket() (pkt av.Packet, err error) {
	if err = self.prepare(); err != nil {
		return
	}
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
		if self.outpkts, err = self.transcoder.Do(rpkt); err != nil {
			return
		}
	}
	return
}

func (self *Demuxer) Streams() (streams []av.CodecData, err error) {
	if err = self.prepare(); err != nil {
		return
	}
	return self.transcoder.Streams()
}

func (self *Demuxer) Close() (err error) {
	if self.transcoder != nil {
		return self.transcoder.Close()
	}
	return
}
