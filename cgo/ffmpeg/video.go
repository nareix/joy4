package ffmpeg

//#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lavresample -lswscale
// #include "ffmpeg.h"
import "C"
import (
	"fmt"
	"image"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/Danile71/joy4/av"
	"github.com/Danile71/joy4/codec/h264parser"
)

type VideoDecoder struct {
	ff        *ffctx
	Extradata []byte
}

func (self *VideoDecoder) Setup() (err error) {
	ff := &self.ff.ff
	if len(self.Extradata) > 0 {
		ff.codecCtx.extradata = (*C.uint8_t)(unsafe.Pointer(&self.Extradata[0]))
		ff.codecCtx.extradata_size = C.int(len(self.Extradata))
	}
	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("ffmpeg: decoder: avcodec_open2 failed")
		return
	}
	return
}

func fromCPtr(buf unsafe.Pointer, size int) (ret []uint8) {
	hdr := (*reflect.SliceHeader)((unsafe.Pointer(&ret)))
	hdr.Cap = size
	hdr.Len = size
	hdr.Data = uintptr(buf)
	return
}

type VideoFrame struct {
	Image image.YCbCr
	Raw   []byte
	Size  int
}

func (self *VideoFrame) Free() {
	self.Image = image.YCbCr{}
	self.Raw = make([]byte, 0)
}

func freeVideoFrame(self *VideoFrame) {
	self.Free()
}

func (self *VideoDecoder) Decode(pkt []byte) (img *VideoFrame, err error) {
	ff := &self.ff.ff

	cgotimg := C.int(0)
	frame := C.av_frame_alloc()
	defer C.av_frame_free(&frame)

	cerr := C.decode(ff.codecCtx, frame, (*C.uchar)(unsafe.Pointer(&pkt[0])), C.int(len(pkt)), &cgotimg)

	if cerr < C.int(0) {
		err = fmt.Errorf("ffmpeg: decode failed: %d", cerr)
		return
	}

	if cgotimg != C.int(0) {
		w := int(frame.width)
		h := int(frame.height)
		ys := int(frame.linesize[0])
		cs := int(frame.linesize[1])

		img = &VideoFrame{Image: image.YCbCr{
			Y:              fromCPtr(unsafe.Pointer(frame.data[0]), ys*h),
			Cb:             fromCPtr(unsafe.Pointer(frame.data[1]), cs*h/2),
			Cr:             fromCPtr(unsafe.Pointer(frame.data[2]), cs*h/2),
			YStride:        ys,
			CStride:        cs,
			SubsampleRatio: image.YCbCrSubsampleRatio420,
			Rect:           image.Rect(0, 0, w, h),
		}}

		runtime.SetFinalizer(img, freeVideoFrame)

		packet := C.AVPacket{}
		defer C.av_packet_unref(&packet)

		cerr := C.avcodec_encode_jpeg(ff.codecCtx, frame, &packet)

		if cerr != C.int(0) {
			err = fmt.Errorf("ffmpeg: avcodec_encode_jpeg failed: %d", cerr)
			return
		}

		img.Size = int(packet.size)
		img.Raw = make([]byte, img.Size)
		copy(img.Raw, *(*[]byte)(unsafe.Pointer(&packet.data)))
	}

	return
}

func (self *VideoDecoder) DecodeBac(pkt []byte) (img *VideoFrame, err error) {
	ff := &self.ff.ff

	cgotimg := C.int(0)
	frame := C.av_frame_alloc()
	defer C.av_frame_free(&frame)

	cerr := C.decode(ff.codecCtx, frame, (*C.uchar)(unsafe.Pointer(&pkt[0])), C.int(len(pkt)), &cgotimg)

	if cerr < C.int(0) {
		err = fmt.Errorf("ffmpeg: avcodec_decode_video2 failed: %d", cerr)
		return
	}

	if cgotimg != C.int(0) {
		w := int(frame.width)
		h := int(frame.height)
		ys := int(frame.linesize[0])
		cs := int(frame.linesize[1])

		img = &VideoFrame{Image: image.YCbCr{
			Y:              fromCPtr(unsafe.Pointer(frame.data[0]), ys*h),
			Cb:             fromCPtr(unsafe.Pointer(frame.data[1]), cs*h/2),
			Cr:             fromCPtr(unsafe.Pointer(frame.data[2]), cs*h/2),
			YStride:        ys,
			CStride:        cs,
			SubsampleRatio: image.YCbCrSubsampleRatio420,
			Rect:           image.Rect(0, 0, w, h),
		}}
		runtime.SetFinalizer(img, freeVideoFrame)

		packet := C.AVPacket{}
		defer C.av_packet_unref(&packet)

		cerr := C.avcodec_encode_jpeg(ff.codecCtx, frame, &packet)

		if cerr != C.int(0) {
			err = fmt.Errorf("ffmpeg: avcodec_encode_jpeg failed: %d", cerr)
			return
		}

		img.Size = int(packet.size)
		img.Raw = make([]byte, img.Size)
		copy(img.Raw, *(*[]byte)(unsafe.Pointer(&packet.data)))
	}

	return
}

func NewVideoDecoder(stream av.CodecData) (dec *VideoDecoder, err error) {
	_dec := &VideoDecoder{}
	var id uint32

	switch stream.Type() {
	case av.H264:
		h264 := stream.(h264parser.CodecData)
		_dec.Extradata = h264.AVCDecoderConfRecordBytes()
		id = C.AV_CODEC_ID_H264

	default:
		err = fmt.Errorf("ffmpeg: NewVideoDecoder codec=%v unsupported", stream.Type())
		return
	}

	c := C.avcodec_find_decoder(id)
	if c == nil || C.avcodec_get_type(id) != C.AVMEDIA_TYPE_VIDEO {
		err = fmt.Errorf("ffmpeg: cannot find video decoder codecId=%d", id)
		return
	}

	if _dec.ff, err = newFFCtxByCodec(c); err != nil {
		return
	}
	if err = _dec.Setup(); err != nil {
		return
	}

	dec = _dec
	return
}
