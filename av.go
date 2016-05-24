package av

type SampleFormat int

const (
	U8 = SampleFormat(iota+1)
	S16
	S32
	FLT
	DBL
	U8P
	S16P
	S32P
	FLTP
	DBLP
	U32
)

func (self SampleFormat) BytesPerSample() int {
	switch self {
	case U8,U8P:
		return 1
	case S16,S16P:
		return 2
	case FLT,FLTP,S32,S32P,U32:
		return 4
	case DBL,DBLP:
		return 8
	default:
		return 0
	}
}

func (self SampleFormat) IsPlanar() bool {
	switch self {
	case S16P,S32P,FLTP,DBLP:
		return true
	default:
		return false
	}
}

const (
	H264 = 0x264
	AAC = 0xaac
)

type CodecData interface {
	IsVideo() bool
	IsAudio() bool
	Type() int
}

type VideoCodecData interface {
	CodecData
	Width() int
	Height() int
}

type AudioCodecData interface {
	CodecData
	SampleFormat() SampleFormat
	SampleRate() int
	ChannelCount() int
}

type H264CodecData interface {
	VideoCodecData
	AVCDecoderConfRecordBytes() []byte
	SPS() []byte
	PPS() []byte
}

type AACCodecData interface {
	AudioCodecData
	MPEG4AudioConfigBytes() []byte
	MakeADTSHeader(samples int, payloadLength int) []byte
}

type Muxer interface {
	WriteHeader([]CodecData) error
	WritePacket(int, Packet) error
	WriteTrailer() error
}

type Demuxer interface {
	ReadPacket() (int, Packet, error)
	Duration() float64
	Streams() []CodecData
}

type Packet struct {
	IsKeyFrame      bool
	Data            []byte
	Duration        float64
	CompositionTime float64
}

type AudioFrame struct {
	SampleFormat SampleFormat
	ChannelCount int
	Bytes []byte
}

type AudioEncoder interface {
	CodecData() AudioCodecData
	Encode(AudioFrame) ([]Packet, error)
	Flush() ([]Packet, error)
}

type AudioDecoder interface {
	Decode(Packet) (AudioFrame, error)
	Flush() (AudioFrame, error)
}

/*
在写入数据包的时候必须严格按照 V-A-A-A-V-A-A-A-.... 顺序，所有包的时间都必须正确
如果有误，跳过错的那一段

cli := rtsp.Open("xxoo")
cli = &av.TranscodeDemuxer{
	Demuxer: cli,
	Transcoders: []Transcoder{ffmpeg.AudioTranscodeTo("aac")},
}

<script class="miniplayer" controls=true autoplay=true minidash-src="stream1" src="//site/minidash.min.js"></script>

minidash.open('src', function(video) {
})

minidash.HandleConn(func(conn *minidash.Conn) {
	conn.RequestSrc
	muxer, err := conn.WriteHeader(cli.Streams())
	muxer.WritePacket()
})

怎样转码
av.Transconder{
	TranscodeHeader(codecData) ok, codecData, error
	TranscodePacket(Packet, flush) []Packet, error
	FlushPacket() []Packet, error
}

decoder := ffmpeg.FindAudioDecoder(AudioCodecData)
decoder := ffmpeg.FindAudioDecoderByName("aac", CodecData)

av.DemuxTranscoder{
	Demuxer Demuxer
	Transconders []Transconder
}
Streams()
ReadPacket()
ClearPacketCache()

怎样混合多个Demuxer
DemuxerMixer{
	demuxer
}
demuxer.FilterStreams()
streams := demuxer.Streams()
streams[0]
*/

