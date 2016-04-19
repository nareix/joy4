package mp4

import (
	"bytes"
	"fmt"
	"github.com/nareix/av"
	"github.com/nareix/mp4/atom"
	"github.com/nareix/mp4/isom"
	"io"
)

type Demuxer struct {
	R io.ReadSeeker

	streams   []*Stream
	movieAtom *atom.Movie
}

func (self *Demuxer) Streams() (streams []av.Stream) {
	for _, stream := range(self.streams) {
		streams = append(streams, stream)
	}
	return
}

func (self *Demuxer) ReadHeader() (err error) {
	var N int64
	var moov *atom.Movie

	if N, err = self.R.Seek(0, 2); err != nil {
		return
	}
	if _, err = self.R.Seek(0, 0); err != nil {
		return
	}

	lr := &io.LimitedReader{R: self.R, N: N}
	for lr.N > 0 {
		var ar *io.LimitedReader

		var cc4 string
		if ar, cc4, err = atom.ReadAtomHeader(lr, ""); err != nil {
			return
		}

		if cc4 == "moov" {
			if moov, err = atom.ReadMovie(ar); err != nil {
				return
			}
		}

		if _, err = atom.ReadDummy(lr, int(ar.N)); err != nil {
			return
		}
	}

	if moov == nil {
		err = fmt.Errorf("'moov' atom not found")
		return
	}
	self.movieAtom = moov

	self.streams = []*Stream{}
	for i, atrack := range moov.Tracks {
		stream := &Stream{
			trackAtom: atrack,
			r:         self.R,
			idx:       i,
		}
		if atrack.Media != nil && atrack.Media.Info != nil && atrack.Media.Info.Sample != nil {
			stream.sample = atrack.Media.Info.Sample
			stream.SetTimeScale(int64(atrack.Media.Header.TimeScale))
		} else {
			err = fmt.Errorf("sample table not found")
			return
		}

		if avc1 := atom.GetAvc1ConfByTrack(atrack); avc1 != nil {
			stream.SetType(av.H264)
			if err = stream.SetCodecData(avc1.Data); err != nil {
				return
			}
			self.streams = append(self.streams, stream)

		} else if mp4a := atom.GetMp4aDescByTrack(atrack); mp4a != nil && mp4a.Conf != nil {
			stream.SetType(av.AAC)
			var config []byte
			if config, err = isom.ReadElemStreamDesc(bytes.NewReader(mp4a.Conf.Data)); err != nil {
				return
			}
			if err = stream.SetCodecData(config); err != nil {
				return
			}
			self.streams = append(self.streams, stream)

		}
	}

	return
}

func (self *Stream) setSampleIndex(index int) (err error) {
	found := false
	start := 0
	self.chunkGroupIndex = 0

	for self.chunkIndex = range self.sample.ChunkOffset.Entries {
		n := self.sample.SampleToChunk.Entries[self.chunkGroupIndex].SamplesPerChunk
		if index >= start && index < start+n {
			found = true
			self.sampleIndexInChunk = index - start
			break
		}
		start += n
		if self.chunkGroupIndex+1 < len(self.sample.SampleToChunk.Entries) &&
			self.chunkIndex+1 == self.sample.SampleToChunk.Entries[self.chunkGroupIndex+1].FirstChunk {
			self.chunkGroupIndex++
		}
	}
	if !found {
		err = fmt.Errorf("stream[%d]: cannot locate sample index in chunk", self.idx)
		return
	}

	if self.sample.SampleSize.SampleSize != 0 {
		self.sampleOffsetInChunk = int64(self.sampleIndexInChunk * self.sample.SampleSize.SampleSize)
	} else {
		if index >= len(self.sample.SampleSize.Entries) {
			err = fmt.Errorf("stream[%d]: sample index out of range", self.idx)
			return
		}
		self.sampleOffsetInChunk = int64(0)
		for i := index - self.sampleIndexInChunk; i < index; i++ {
			self.sampleOffsetInChunk += int64(self.sample.SampleSize.Entries[i])
		}
	}

	self.dts = int64(0)
	start = 0
	found = false
	self.sttsEntryIndex = 0
	for self.sttsEntryIndex < len(self.sample.TimeToSample.Entries) {
		entry := self.sample.TimeToSample.Entries[self.sttsEntryIndex]
		n := entry.Count
		if index >= start && index < start+n {
			self.sampleIndexInSttsEntry = index - start
			self.dts += int64((index - start) * entry.Duration)
			found = true
			break
		}
		start += n
		self.dts += int64(n * entry.Duration)
		self.sttsEntryIndex++
	}
	if !found {
		err = fmt.Errorf("stream[%d]: cannot locate sample index in stts entry", self.idx)
		return
	}

	if self.sample.CompositionOffset != nil && len(self.sample.CompositionOffset.Entries) > 0 {
		start = 0
		found = false
		self.cttsEntryIndex = 0
		for self.cttsEntryIndex < len(self.sample.CompositionOffset.Entries) {
			n := self.sample.CompositionOffset.Entries[self.cttsEntryIndex].Count
			if index >= start && index < start+n {
				self.sampleIndexInCttsEntry = index - start
				found = true
				break
			}
			start += n
			self.cttsEntryIndex++
		}
		if !found {
			err = fmt.Errorf("stream[%d]: cannot locate sample index in ctts entry", self.idx)
			return
		}
	}

	if self.sample.SyncSample != nil {
		self.syncSampleIndex = 0
		for self.syncSampleIndex < len(self.sample.SyncSample.Entries)-1 {
			if self.sample.SyncSample.Entries[self.syncSampleIndex+1]-1 > index {
				break
			}
			self.syncSampleIndex++
		}
	}

	self.sampleIndex = index
	return
}

func (self *Stream) isSampleValid() bool {
	if self.chunkIndex >= len(self.sample.ChunkOffset.Entries) {
		return false
	}
	if self.chunkGroupIndex >= len(self.sample.SampleToChunk.Entries) {
		return false
	}
	if self.sttsEntryIndex >= len(self.sample.TimeToSample.Entries) {
		return false
	}
	if self.sample.CompositionOffset != nil && len(self.sample.CompositionOffset.Entries) > 0 {
		if self.cttsEntryIndex >= len(self.sample.CompositionOffset.Entries) {
			return false
		}
	}
	if self.sample.SyncSample != nil {
		if self.syncSampleIndex >= len(self.sample.SyncSample.Entries) {
			return false
		}
	}
	if self.sample.SampleSize.SampleSize != 0 {
		if self.sampleIndex >= len(self.sample.SampleSize.Entries) {
			return false
		}
	}
	return true
}

func (self *Stream) incSampleIndex() {
	self.sampleIndexInChunk++
	if self.sampleIndexInChunk == self.sample.SampleToChunk.Entries[self.chunkGroupIndex].SamplesPerChunk {
		self.chunkIndex++
		self.sampleIndexInChunk = 0
		self.sampleOffsetInChunk = int64(0)
	} else {
		if self.sample.SampleSize.SampleSize != 0 {
			self.sampleOffsetInChunk += int64(self.sample.SampleSize.SampleSize)
		} else {
			self.sampleOffsetInChunk += int64(self.sample.SampleSize.Entries[self.sampleIndex])
		}
	}

	if self.chunkGroupIndex+1 < len(self.sample.SampleToChunk.Entries) &&
		self.chunkIndex+1 == self.sample.SampleToChunk.Entries[self.chunkGroupIndex+1].FirstChunk {
		self.chunkGroupIndex++
	}

	sttsEntry := self.sample.TimeToSample.Entries[self.sttsEntryIndex]
	self.sampleIndexInSttsEntry++
	self.dts += int64(sttsEntry.Duration)
	if self.sampleIndexInSttsEntry == sttsEntry.Count {
		self.sampleIndexInSttsEntry = 0
		self.sttsEntryIndex++
	}

	if self.sample.CompositionOffset != nil && len(self.sample.CompositionOffset.Entries) > 0 {
		self.sampleIndexInCttsEntry++
		if self.sampleIndexInCttsEntry == self.sample.CompositionOffset.Entries[self.cttsEntryIndex].Count {
			self.sampleIndexInCttsEntry = 0
			self.cttsEntryIndex++
		}
	}

	if self.sample.SyncSample != nil {
		entries := self.sample.SyncSample.Entries
		if self.syncSampleIndex+1 < len(entries) && entries[self.syncSampleIndex+1]-1 == self.sampleIndex+1 {
			self.syncSampleIndex++
		}
	}

	self.sampleIndex++
}

func (self *Stream) sampleCount() int {
	if self.sample.SampleSize.SampleSize == 0 {
		chunkGroupIndex := 0
		count := 0
		for chunkIndex := range self.sample.ChunkOffset.Entries {
			n := self.sample.SampleToChunk.Entries[chunkGroupIndex].SamplesPerChunk
			count += n
			if chunkGroupIndex+1 < len(self.sample.SampleToChunk.Entries) &&
				chunkIndex+1 == self.sample.SampleToChunk.Entries[chunkGroupIndex+1].FirstChunk {
				chunkGroupIndex++
			}
		}
		return count
	} else {
		return len(self.sample.SampleSize.Entries)
	}
}

func (self *Demuxer) ReadPacket() (pkt av.Packet, err error) {
	var choose *Stream
	for _, stream := range(self.streams) {
		if choose == nil || stream.TsToTime(stream.dts) < choose.TsToTime(choose.dts) {
			choose = stream
		}
	}
	if false {
		fmt.Printf("ReadPacket: choose index=%v time=%v\n", choose.idx, choose.TsToTime(choose.dts))
	}
	pkt.StreamIdx = choose.idx
	pkt.Pts, pkt.Dts, pkt.IsKeyFrame, pkt.Data, err = choose.readSample()
	return
}

func (self *Demuxer) SeekToTime(time float64) (err error) {
	for _, stream := range(self.streams) {
		if stream.IsVideo() {
			if err = stream.seekToTime(time); err != nil {
				return
			}
			time = stream.TsToTime(stream.dts)
			break
		}
	}

	for _, stream := range(self.streams) {
		if !stream.IsVideo() {
			if err = stream.seekToTime(time); err != nil {
				return
			}
		}
	}

	return
}

func (self *Stream) readSample() (pts int64, dts int64, isKeyFrame bool, data []byte, err error) {
	if !self.isSampleValid() {
		err = io.EOF
		return
	}

	chunkOffset := self.sample.ChunkOffset.Entries[self.chunkIndex]
	sampleSize := 0
	if self.sample.SampleSize.SampleSize != 0 {
		sampleSize = self.sample.SampleSize.SampleSize
	} else {
		sampleSize = self.sample.SampleSize.Entries[self.sampleIndex]
	}

	sampleOffset := int64(chunkOffset) + self.sampleOffsetInChunk
	if _, err = self.r.Seek(int64(sampleOffset), 0); err != nil {
		return
	}

	data = make([]byte, sampleSize)
	if _, err = self.r.Read(data); err != nil {
		return
	}

	if self.sample.SyncSample != nil {
		if self.sample.SyncSample.Entries[self.syncSampleIndex]-1 == self.sampleIndex {
			isKeyFrame = true
		}
	}

	//println("pts/dts", self.ptsEntryIndex, self.dtsEntryIndex)
	dts = self.dts
	if self.sample.CompositionOffset != nil && len(self.sample.CompositionOffset.Entries) > 0 {
		pts = self.dts + int64(self.sample.CompositionOffset.Entries[self.cttsEntryIndex].Offset)
	} else {
		pts = dts
	}

	self.incSampleIndex()
	return
}

func (self *Stream) duration() float64 {
	total := int64(0)
	for _, entry := range self.sample.TimeToSample.Entries {
		total += int64(entry.Duration * entry.Count)
	}
	return float64(total) / float64(self.TimeScale())
}

func (self *Stream) seekToTime(time float64) (err error) {
	index := self.timeToSampleIndex(time)
	if err = self.setSampleIndex(index); err != nil {
		return
	}
	if false {
		fmt.Printf("stream[%d]: seekToTime index=%v time=%v cur=%v\n", self.idx, index, time, self.TsToTime(self.dts))
	}
	return
}

func (self *Stream) timeToSampleIndex(time float64) int {
	targetTs := self.TimeToTs(time)
	targetIndex := 0

	startTs := int64(0)
	endTs := int64(0)
	startIndex := 0
	endIndex := 0
	found := false
	for _, entry := range self.sample.TimeToSample.Entries {
		endTs = startTs + int64(entry.Count*entry.Duration)
		endIndex = startIndex + entry.Count
		if targetTs >= startTs && targetTs < endTs {
			targetIndex = startIndex + int((targetTs-startTs)/int64(entry.Duration))
			found = true
		}
		startTs = endTs
		startIndex = endIndex
	}
	if !found {
		if targetTs < 0 {
			targetIndex = 0
		} else {
			targetIndex = endIndex - 1
		}
	}

	if self.sample.SyncSample != nil {
		entries := self.sample.SyncSample.Entries
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i]-1 < targetIndex {
				targetIndex = entries[i] - 1
				break
			}
		}
	}

	return targetIndex
}

