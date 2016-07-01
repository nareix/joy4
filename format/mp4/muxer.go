package mp4

import (
	"bytes"
	"fmt"
	"time"
	"github.com/nareix/joy4/av"
	"github.com/nareix/pio"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/format/mp4/atom"
	"github.com/nareix/joy4/format/mp4/isom"
	"io"
)

type Muxer struct {
	W          io.WriteSeeker
	streams    []*Stream
	mdatWriter *atom.Writer
}

func (self *Muxer) isCodecSupported(codec av.CodecData) bool {
	switch codec.Type() {
	case av.H264, av.AAC:
		return true
	default:
		return false
	}
}

func (self *Muxer) newStream(codec av.CodecData) (err error) {
	if !self.isCodecSupported(codec) {
		err = fmt.Errorf("mp4: codec type=%v is not supported", codec.Type())
		return
	}

	stream := &Stream{CodecData: codec}

	stream.sample = &atom.SampleTable{
		SampleDesc:   &atom.SampleDesc{},
		TimeToSample: &atom.TimeToSample{},
		SampleToChunk: &atom.SampleToChunk{
			Entries: []atom.SampleToChunkEntry{
				{
					FirstChunk:      1,
					SampleDescId:    1,
					SamplesPerChunk: 1,
				},
			},
		},
		SampleSize:  &atom.SampleSize{},
		ChunkOffset: &atom.ChunkOffset{},
	}

	stream.trackAtom = &atom.Track{
		Header: &atom.TrackHeader{
			TrackId:  len(self.streams) + 1,
			Flags:    0x0003, // Track enabled | Track in movie
			Duration: 0,      // fill later
			Matrix:   [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		},
		Media: &atom.Media{
			Header: &atom.MediaHeader{
				TimeScale: 0, // fill later
				Duration:  0, // fill later
				Language:  21956,
			},
			Info: &atom.MediaInfo{
				Sample: stream.sample,
				Data: &atom.DataInfo{
					Refer: &atom.DataRefer{
						Url: &atom.DataReferUrl{
							Flags: 0x000001, // Self reference
						},
					},
				},
			},
		},
	}

	switch codec.Type() {
	case av.H264:
		stream.sample.SyncSample = &atom.SyncSample{}
	}

	stream.timeScale = 90000
	stream.muxer = self
	self.streams = append(self.streams, stream)

	return
}

func (self *Stream) fillTrackAtom() (err error) {
	self.trackAtom.Media.Header.TimeScale = int(self.timeScale)
	self.trackAtom.Media.Header.Duration = int(self.duration)

	if self.Type() == av.H264 {
		codec := self.CodecData.(h264parser.CodecData)
		width, height := codec.Width(), codec.Height()
		self.sample.SampleDesc.Avc1Desc = &atom.Avc1Desc{
			DataRefIdx:           1,
			HorizontalResolution: 72,
			VorizontalResolution: 72,
			Width:                int(width),
			Height:               int(height),
			FrameCount:           1,
			Depth:                24,
			ColorTableId:         -1,
			Conf:                 &atom.Avc1Conf{Data: codec.AVCDecoderConfRecordBytes()},
		}
		self.trackAtom.Media.Handler = &atom.HandlerRefer{
			SubType: "vide",
			Name:    "Video Media Handler",
		}
		self.trackAtom.Media.Info.Video = &atom.VideoMediaInfo{
			Flags: 0x000001,
		}
		self.trackAtom.Header.TrackWidth = atom.IntToFixed(int(width))
		self.trackAtom.Header.TrackHeight = atom.IntToFixed(int(height))

	} else if self.Type() == av.AAC {
		codec := self.CodecData.(aacparser.CodecData)
		buf := &bytes.Buffer{}
		if err = isom.WriteElemStreamDesc(buf, codec.MPEG4AudioConfigBytes(), uint(self.trackAtom.Header.TrackId)); err != nil {
			return
		}
		self.sample.SampleDesc.Mp4aDesc = &atom.Mp4aDesc{
			DataRefIdx:       1,
			NumberOfChannels: codec.ChannelLayout().Count(),
			SampleSize:       codec.SampleFormat().BytesPerSample(),
			SampleRate:       atom.IntToFixed(codec.SampleRate()),
			Conf: &atom.ElemStreamDesc{
				Data: buf.Bytes(),
			},
		}
		self.trackAtom.Header.Volume = atom.IntToFixed(1)
		self.trackAtom.Header.AlternateGroup = 1
		self.trackAtom.Media.Handler = &atom.HandlerRefer{
			SubType: "soun",
			Name:    "Sound Handler",
		}
		self.trackAtom.Media.Info.Sound = &atom.SoundMediaInfo{}

	} else {
		err = fmt.Errorf("mp4: codec type=%d invalid", self.Type())
	}

	return
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	self.streams = []*Stream{}
	for _, stream := range streams {
		if err = self.newStream(stream); err != nil {
			return
		}
	}

	if self.mdatWriter, err = atom.WriteAtomHeader(self.W, "mdat"); err != nil {
		return
	}
	for _, stream := range self.streams {
		if stream.Type().IsVideo() {
			stream.sample.CompositionOffset = &atom.CompositionOffset{}
		}
	}
	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	stream := self.streams[pkt.Idx]
	if err = stream.writePacket(pkt); err != nil {
		return
	}
	return
}

func (self *Stream) writePacket(pkt av.Packet) (err error) {
	if self.lasttime == 0 {
		self.lasttime = pkt.Time
		return
	}
	rawdur := pkt.Time - self.lasttime
	if rawdur < 0 {
		err = fmt.Errorf("mp4: stream#%d time=%v < lasttime=%v", pkt.Idx, pkt.Time, self.lasttime)
		return
	}
	self.lasttime = pkt.Time

	var filePos int64
	var sampleSize int

	if filePos, err = self.muxer.mdatWriter.Seek(0, 1); err != nil {
		return
	}

	if self.Type() == av.H264 {
		if typ := h264parser.CheckNALUsType(pkt.Data); typ != h264parser.NALU_RAW {
			err = fmt.Errorf("mp4: nalu format=%d is not raw", typ)
			return
		}
		var b [4]byte
		pio.PutU32BE(b[:], uint32(len(pkt.Data)))
		sampleSize += len(pkt.Data)+4
		if _, err = self.muxer.mdatWriter.Write(b[:]); err != nil {
			return
		}
		if _, err = self.muxer.mdatWriter.Write(pkt.Data); err != nil {
			return
		}
	} else {
		sampleSize = len(pkt.Data)
		if _, err = self.muxer.mdatWriter.Write(pkt.Data); err != nil {
			return
		}
	}

	if pkt.IsKeyFrame && self.sample.SyncSample != nil {
		self.sample.SyncSample.Entries = append(self.sample.SyncSample.Entries, self.sampleIndex+1)
	}

	duration := int(self.timeToTs(rawdur))
	if self.sttsEntry == nil || duration != self.sttsEntry.Duration {
		self.sample.TimeToSample.Entries = append(self.sample.TimeToSample.Entries, atom.TimeToSampleEntry{Duration: duration})
		self.sttsEntry = &self.sample.TimeToSample.Entries[len(self.sample.TimeToSample.Entries)-1]
	}
	self.sttsEntry.Count++

	if self.sample.CompositionOffset != nil {
		offset := int(self.timeToTs(pkt.CompositionTime))
		if self.cttsEntry == nil || offset != self.cttsEntry.Offset {
			table := self.sample.CompositionOffset
			table.Entries = append(table.Entries, atom.CompositionOffsetEntry{Offset: offset})
			self.cttsEntry = &table.Entries[len(table.Entries)-1]
		}
		self.cttsEntry.Count++
	}

	self.duration += int64(duration)
	self.sampleIndex++
	self.sample.ChunkOffset.Entries = append(self.sample.ChunkOffset.Entries, int(filePos))
	self.sample.SampleSize.Entries = append(self.sample.SampleSize.Entries, sampleSize)

	return
}

func (self *Muxer) WriteTrailer() (err error) {
	moov := &atom.Movie{}
	moov.Header = &atom.MovieHeader{
		PreferredRate:   atom.IntToFixed(1),
		PreferredVolume: atom.IntToFixed(1),
		Matrix:          [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		NextTrackId:     2,
	}

	maxDur := time.Duration(0)
	timeScale := int64(10000)
	for _, stream := range self.streams {
		if err = stream.fillTrackAtom(); err != nil {
			return
		}
		dur := stream.tsToTime(stream.duration)
		stream.trackAtom.Header.Duration = int(timeToTs(dur, timeScale))
		if dur > maxDur {
			maxDur = dur
		}
		moov.Tracks = append(moov.Tracks, stream.trackAtom)
	}
	moov.Header.TimeScale = int(timeScale)
	moov.Header.Duration = int(timeToTs(maxDur, timeScale))

	if err = self.mdatWriter.Close(); err != nil {
		return
	}
	if err = atom.WriteMovie(self.W, moov); err != nil {
		return
	}

	return
}
