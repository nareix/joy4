
package mp4

import (
	"./atom"
	"os"
	"io"
	"log"
	"encoding/hex"
)

type File struct {
}

func (self *File) AddAvcc(avcc *Avcc) {
}

func (self *File) AddMp4a(mp4a *Mp4a) {
}

func (self *File) GetAvcc() (avcc []*Avcc) {
	return
}

func (self *File) GetMp4a() (mp4a []*Mp4a) {
	return
}

func (self *File) Sync() {
}

func (self *File) Close() {
}

func analyzeSamples(sample *atom.SampleTable) {
	log.Println("sample:")

	log.Println("HasCompositionOffset", sample.CompositionOffset != nil)
	log.Println("SampleCount", len(sample.SampleSize.Entries))
	log.Println("ChunkCount", len(sample.ChunkOffset.Entries))

	log.Println("SampleToChunkCount", len(sample.SampleToChunk.Entries))
	log.Println("SampleToChunk[0]", sample.SampleToChunk.Entries[0])
	log.Println("SampleToChunk[1]", sample.SampleToChunk.Entries[1])

	log.Println("TimeToSampleCount", len(sample.TimeToSample.Entries))
	log.Println("TimeToSample[0]", sample.TimeToSample.Entries[0])

	if sample.SyncSample != nil {
		log.Println("SyncSampleCount", len(sample.SyncSample.Entries))
		for i, val := range sample.SyncSample.Entries {
			if i < 5 {
				log.Println("SyncSample", i, val)
			}
		}
		log.Println("...")
	} else {
		log.Println("NoSyncSample")
	}
}

func changeMoov(moov *atom.Movie) {
	header := moov.Header

	header.CreateTime = atom.TimeStamp(0)
	header.ModifyTime = atom.TimeStamp(0)

	if true {
		//log.Println("moov: ", header.CreateTime, header.TimeScale, header.Duration)
		log.Println("moov: ", header.PreferredRate, header.PreferredVolume)
		//log.Println("moov: ", header.PreviewTime, header.PreviewDuration)
		//log.Println("moov: ", header.PosterTime)
		//log.Println("moov: ", header.SelectionTime, header.SelectionDuration)
		//log.Println("moov: ", header.CurrentTime)
		//log.Println("moov: ", header.NextTrackId)
		//log.Println("moov: ", header.Matrix)
	}

	for i, track := range moov.Tracks {
		if true {
			log.Println("track", i, ":", track.Header.TrackId)
			log.Println("track", i, ":", track.Header.Duration)
			//log.Println("track", i, ":", track.Header.Layer, track.Header.AlternateGroup)
			log.Println("track", i, ":", track.Header.Volume)
			log.Println("track", i, ":", track.Header.TrackWidth, track.Header.TrackHeight)
			log.Println("track", i, ":", track.Header.Matrix)
		}

		media := track.Media
		if true {
			log.Println("mediaHeader", media.Header)
			log.Println("media.hdlr", media.Handler)
		}

		if minf := media.Info; minf != nil {

			if true {
				log.Println("minf.video", minf.Video)
			}

			if sample := minf.Sample; sample != nil {
				analyzeSamples(sample)

				if desc := sample.SampleDesc; desc != nil {

					if avc1Desc := desc.Avc1Desc; avc1Desc != nil {
						if conf := avc1Desc.Conf; conf != nil {
							if true {
								//log.Println("avc1", hex.Dump(conf.Data))
								log.Println("avc1desc", conf)
								//avcconf, _ := atom.ReadAVCDecoderConfRecord(bytes.NewReader(conf.Data))
								//log.Println("avcconf", avcconf)
							}
						}
					}

					if mp4a := desc.Mp4aDesc; mp4a != nil {
						if conf := mp4a.Conf; conf != nil {
							if false {
							log.Println("mp4a", hex.Dump(conf.Data))
						}
						}
					}

				}
			}
		}

	}

}

type Sample struct {
	Time int
	Data []byte
	Sync bool
}

func readSamples(vsample *atom.SampleTable, mdat io.ReadSeeker, out chan<- Sample) {
	sampleToChunkIdx := 0
	chunkIdx := 0
	nextChunkIdx := 0
	samplesPerChunk := 0

	updateSamplesPerChunk := func() {
		chunkIdx = vsample.SampleToChunk.Entries[sampleToChunkIdx].FirstChunk-1
		samplesPerChunk = vsample.SampleToChunk.Entries[sampleToChunkIdx].SamplesPerChunk
		sampleToChunkIdx++
		if sampleToChunkIdx < len(vsample.SampleToChunk.Entries) {
			nextChunkIdx = vsample.SampleToChunk.Entries[sampleToChunkIdx].FirstChunk-1
		} else {
			nextChunkIdx = -1
		}
	}
	updateSamplesPerChunk()

	timeToSampleIdx := 0
	timeToSampleCount := 0
	sampleTime := 0

	sampleIdx := 0
	sampleNr := len(vsample.SampleSize.Entries)

	syncSampleIdx := 0
	syncSample := vsample.SyncSample.Entries;

	for sampleIdx < sampleNr {
		if chunkIdx == nextChunkIdx {
			updateSamplesPerChunk()
		}
		sampleOffset := vsample.ChunkOffset.Entries[chunkIdx]
		for i := 0; i < samplesPerChunk; i++ {
			sampleSize := vsample.SampleSize.Entries[sampleIdx]

			mdat.Seek(int64(sampleOffset), 0)
			data := make([]byte, sampleSize)
			mdat.Read(data)

			var sync bool
			if syncSampleIdx < len(syncSample) && syncSample[syncSampleIdx]-1 == sampleIdx {
				sync = true
				syncSampleIdx++
			}

			out <- Sample{
				Time: sampleTime,
				Data: data,
				Sync: sync,
			}

			sampleOffset += sampleSize
			sampleIdx++

			sampleTime += vsample.TimeToSample.Entries[timeToSampleIdx].Duration
			timeToSampleCount++
			if timeToSampleCount == vsample.TimeToSample.Entries[timeToSampleIdx].Count {
				timeToSampleCount = 0
				timeToSampleIdx++
			}
		}
		chunkIdx++
	}

	close(out)
}

func rewrite(moov *atom.Movie, mdat io.ReadSeeker, outfile io.WriteSeeker) (err error) {
	var vtrack *atom.Track
	var vsample *atom.SampleTable

	for _, track := range moov.Tracks {
		media := track.Media
		if minf := media.Info; minf != nil {
			if sample := minf.Sample; sample != nil {
				if desc := sample.SampleDesc; desc != nil {
					if avc1Desc := desc.Avc1Desc; avc1Desc != nil {
						if conf := avc1Desc.Conf; conf != nil {
							vtrack = track
							vsample = sample
						}
					}
				}
			}
		}
	}

	sampleCh := make(chan Sample)
	go readSamples(vsample, mdat, sampleCh)

	log.Println("avc1Desc.conf", vsample.SampleDesc.Avc1Desc.Conf)

	newsample := &atom.SampleTable{
		SampleDesc: &atom.SampleDesc{
			Avc1Desc: &atom.Avc1Desc{
				Conf: vsample.SampleDesc.Avc1Desc.Conf,
			},
		},
		TimeToSample: &atom.TimeToSample{},
		SampleToChunk: &atom.SampleToChunk{
			Entries: []atom.SampleToChunkEntry{
				{
					FirstChunk: 1,
					SampleDescId: 1,
				},
			},
		},
		SampleSize: &atom.SampleSize{},
		ChunkOffset: &atom.ChunkOffset{
			Entries: []int{8},
		},
		SyncSample: &atom.SyncSample{},
	}
	sampleToChunk := &newsample.SampleToChunk.Entries[0]

	var timeToSample *atom.TimeToSampleEntry

	mdatWriter, _ := atom.WriteAtomHeader(outfile, "mdat")

	for sampleIdx := 1; ; sampleIdx++ {
		if sample, ok := <-sampleCh; ok {
			if sampleIdx < 10 {
				log.Println(
					sampleIdx,
					"sampleTime", float32(sample.Time)/float32(vtrack.Media.Header.TimeScale)/60.0,
					"len", len(sample.Data),
					//"timeToSampleIdx", timeToSampleIdx,
				)
			}

			sampleSize := len(sample.Data)
			sampleDuration := 1000
			mdatWriter.Write(sample.Data)

			if sample.Sync {
				newsample.SyncSample.Entries = append(newsample.SyncSample.Entries, sampleIdx)
			}

			if timeToSample != nil && sampleDuration != timeToSample.Duration {
				newsample.TimeToSample.Entries = append(newsample.TimeToSample.Entries, *timeToSample)
				timeToSample = nil
			}
			if timeToSample == nil {
				timeToSample = &atom.TimeToSampleEntry{
					Duration: sampleDuration,
				}
			}
			timeToSample.Count++

			sampleToChunk.SamplesPerChunk++

			newsample.SampleSize.Entries = append(newsample.SampleSize.Entries, sampleSize)
		} else {
			break
		}
	}

	if timeToSample != nil {
		newsample.TimeToSample.Entries = append(newsample.TimeToSample.Entries, *timeToSample)
	}

	mdatWriter.Close()

	newmoov := &atom.Movie{}
	newmoov.Header = &atom.MovieHeader{
		TimeScale: moov.Header.TimeScale,
		Duration: moov.Header.Duration,
		PreferredRate: moov.Header.PreferredRate,
		PreferredVolume: moov.Header.PreferredVolume,
		Matrix: [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
	}

	newtrack := &atom.Track{
		Header: &atom.TrackHeader{
			Flags: 0x0001, // enabled
			Duration: vtrack.Header.Duration,
			Volume: vtrack.Header.Volume,
			Matrix: [9]int{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
			//TrackWidth: vtrack.Header.TrackWidth,
			//TrackHeight: vtrack.Header.TrackHeight,
			TrackId: 1,
		},

		Media: &atom.Media{
			Header: &atom.MediaHeader{
				TimeScale: vtrack.Media.Header.TimeScale,
				Duration: vtrack.Media.Header.Duration,
			},
			Info: &atom.MediaInfo{
				Video: &atom.VideoMediaInfo{
					Flags: 0x000001,
				},
				Sample: newsample,
			},
			Handler: &atom.HandlerRefer{
				SubType: "vide",
				Name: "Video Media Handler",
			},
		},
	}
	newmoov.Tracks = append(newmoov.Tracks, newtrack)

	atom.WriteMovie(outfile, newmoov)

	return
}

func TestRewrite(filename string) (file *File, err error) {
	var infile *os.File
	if infile, err = os.Open(filename); err != nil {
		return
	}

	var finfo os.FileInfo
	if finfo, err = infile.Stat(); err != nil {
		return
	}
	lr := &io.LimitedReader{R: infile, N: finfo.Size()}

	var moov *atom.Movie
	mdatOffset := int64(-1)

	for lr.N > 0 {
		var ar *io.LimitedReader

		var cc4 string
		if ar, cc4, err = atom.ReadAtomHeader(lr, ""); err != nil {
			return
		}
		log.Println("cc4", cc4)

		if cc4 == "moov" {
			if moov, err = atom.ReadMovie(ar); err != nil {
				return
			}
		} else if cc4 == "mdat" {
			mdatOffset, _ = infile.Seek(0, 1)
			break
		}

		if _, err = atom.ReadDummy(lr, int(ar.N)); err != nil {
			return
		}
	}

	if mdatOffset == -1 {
		log.Println("mdat not found")
		return
	}

	outfileName := filename+".out.mp4"
	var outfile *os.File
	if outfile, err = os.Create(outfileName); err != nil {
		return
	}

	if err = rewrite(moov, infile, outfile); err != nil {
		return
	}

	if err = outfile.Close(); err != nil {
		return
	}
	log.Println("output file", outfileName, "saved")

	return
}

func TestConvert(filename string) (file *File, err error) {
	var osfile *os.File
	if osfile, err = os.Open(filename); err != nil {
		return
	}

	var finfo os.FileInfo
	if finfo, err = osfile.Stat(); err != nil {
		return
	}
	log.Println("filesize", finfo.Size())

	lr := &io.LimitedReader{R: osfile, N: finfo.Size()}

	var outfile *os.File
	if outfile, err = os.Create(filename+".out.mp4"); err != nil {
		return
	}

	for lr.N > 0 {
		var ar *io.LimitedReader

		var cc4 string
		if ar, cc4, err = atom.ReadAtomHeader(lr, ""); err != nil {
			return
		}

		if cc4 == "moov" {

			curPos, _ := outfile.Seek(0, 1)
			origSize := ar.N+8
			var moov *atom.Movie
			if moov, err = atom.ReadMovie(ar); err != nil {
				return
			}
			changeMoov(moov)
			if err = atom.WriteMovie(outfile, moov); err != nil {
				return
			}
			curPosAfterRead, _ := outfile.Seek(0, 1)
			bytesWritten := curPosAfterRead - curPos

			log.Println("regen moov", "tracks nr", len(moov.Tracks),
				"origSize", origSize, "bytesWritten", bytesWritten,
			)

			padSize := origSize - bytesWritten - 8
			aw, _ := atom.WriteAtomHeader(outfile, "free")
			atom.WriteDummy(outfile, int(padSize))
			aw.Close()

		} else {

			var outcc4 string
			if cc4 != "mdat" {
				outcc4 = "free"
			} else {
				outcc4 = "mdat"
			}
			var aw *atom.Writer
			if aw, err = atom.WriteAtomHeader(outfile, outcc4); err != nil {
				return
			}
			log.Println("copy", cc4)
			if _, err = io.CopyN(aw, ar, ar.N); err != nil {
				return
			}
			if err = aw.Close(); err != nil {
				return
			}
		}

		//log.Println("atom", cc4, "left", lr.N)
		//atom.ReadDummy(ar, int(ar.N))
	}

	if err = outfile.Close(); err != nil {
		return
	}

	return
}

func Create(filename string) (file *File, err error) {
	return
}

