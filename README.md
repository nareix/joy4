
joy4: A modern multimedia library for Golang
==

* Well-designed API, easy to use, write a media server in 1 minute:

  ```go
  import (
    "github.com/nareix/av"
  )

  func main() {
    demuxer, _ := rtsp.Open("rtsp://xxooxo")
    outfile, _ := os.Create("out.mp4")
    muxer := mp4.Create(outfile, demuxer.Streams())
    av.CopyPackets(muxer, demuxer)
    muxer.WriteTrailer()
    outfile.Close()
  }
  ```

  ​


* Support MPEG-TS/MP4/FLV muxer/demuxer and RTSP client
* Support H264/AAC parser/encoder/decoder and transcode (cgo ffmpeg binding)


* high performance RTSP/RTMP/HLS/MPEG-DASH streaming server
* AV Database: store audio/video packets 



# 简单易用的 Golang 流媒体框架

* 简单易用的 API，一分钟内可以实现很多东西

* 支持 MPEG-TS/MP4/FLV 格式，以及 RTSP/RTMP 客户端

* 支持 H264/AAC 编解码以及转码（使用 cgo 调用 ffmpeg）

* 支持输入输出设备

* 高性能 RTMP/FLV/HLS/MPEG-DASH 服务器，支持 OBS/ffmpeg 推流，支持 CDN 边缘节点模式

* 音视频数据库：支持录播/点播。支持分布式部署

* 与其他流行流媒体框架的对比以及适用场景
  * ffmpeg 是目前支持格式最全使用最广泛的编解码库，本框架以 Golang 的方式封装了 ffmpeg 的编解码部分，开发更方便

  * live555 是 C++ 开发的流媒体框架，支持 RTSP，不支持新的流媒体协议，开发难度较大

  * gstreamer 是 C 开发的流媒体框架，功能齐全，包含编解码/画面声音处理/转码。但是开发难度极大，对各插件的行为很难控制，需要学习过多的概念，除了直接调用命令行的简单应用之外，很难用在实际项目中

  * nginx-rtmp 是基于 nginx 开发的 RTMP/HLS/MPEG-DASH 服务器，性能很强。但是它是一个现成的服务器不是一个库，很难利用它的代码进行二次开发

  * av314 采用 Golang 语言编写，与 C/C++ 开发的其他框架相比，在保持高性能的同时没有 C/C++ 的各种问题，API 非常简单易用

  * av314 支持最新的流媒体格式，对于旧的格式也能使用 ffmpeg 来支持

  * av314 支持 Windows/Mac/Linux/树莓派

    ​

示例：

- 读取文件
- 读取文件然后格式转换保存文件
- 新建一个 RTSP Server 并在请求时候播放文件
- 从 RTMP Client 读取流并转移保存到文件
- RTMP 推流并建立



```go
dec := h264dec.New(CodecData)
dec.Write(nalu)
dec.ReadFrame()

enc := h264enc.New()

stream := NewStream(type, codecData)

new := &av.Transcoder{
  R: demuxer,
}
new.FindEncoder = func(type av.CodecType, codecData []byte) {
  if type == av.G722 {
  	return av.AAC, aacenc.New(codecData)
  }
}
new.ReadPacket()
```



streams, _ := muxer.ReadHeader()