# A pure golang mp4 library

Provides mp4 reader/writer and mp4 atom manipulations functions.

Open a mp4 file and read the first sample:
```go
file, _ := os.Open("test.mp4")
demuxer := &mp4.Demuxer{R: file}
demuxer.ReadHeader()
pts, dts, isKeyFrame, data, err := demuxer.TrackH264.ReadSample()
```

do some seeking:

```go
demuxer.TrackH264.SeekToTime(2.0)
```

demuxer demo code [here](https://github.com/nareix/mp4/blob/master/example/example.go#L11)

the library also provide atom struct decoding/encoding functions(
learn more about mp4 atoms [here](https://developer.apple.com/library/mac/documentation/QuickTime/QTFF/QTFFChap2/qtff2.html)
)

you can access atom structs via `Demuxer.TrackH264.TrackAtom`. for example:

```go
// Get the raw TimeScale field inside `mvhd` atom
fmt.Println(demuxer.TrackH264.TrackAtom.Media.Header.TimeScale)
```

for more see Atom API Docs

# Documentation

[API Docs](https://godoc.org/github.com/nareix/mp4)

[Atom API Docs](https://godoc.org/github.com/nareix/mp4/atom)
