package ffmpeg

/*
#include "ffmpeg.h"
int wrap_avcodec_decode_video2(AVCodecContext *ctx, AVFrame *frame, void *data, int size, int *got) {
	struct AVPacket pkt = {.data = data, .size = size};
	return avcodec_decode_video2(ctx, frame, got, &pkt);
}
*/
import "C"
import (
	"unsafe"
	"runtime"
	"fmt"
	"image"
	"reflect"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
)


// VideoEncoder contains all params that must be set by user to initialize the video encoder
type VideoEncoder struct {
	ff *ffctx
	Bitrate int
	width int
	height int
	gopSize int
	fpsNum, fpsDen int
	pixelFormat av.PixelFormat
	codecData h264parser.CodecData
	codecDataInitialised bool
	pts int64
}

// Setup initializes the encoder context and checks user params
func (enc *VideoEncoder) Setup() (err error) {
	ff := &enc.ff.ff

	ff.frame = C.av_frame_alloc()

	// TODO check params (bitrate, etc)

	// if enc.PixelFormat == av.PixelFormat(0) {
	// 	enc.PixelFormat = PixelFormatFF2AV(*ff.codec.sample_fmts)
	// }

	//if enc.Bitrate == 0 {
	//	enc.Bitrate = 80000
	//}

	// All the following params are described in ffmpeg: avcodec.h, in struct AVCodecContext
	ff.codecCtx.width			= C.int(enc.width)
	ff.codecCtx.height			= C.int(enc.height)
	ff.codecCtx.pix_fmt			= PixelFormatAV2FF(enc.pixelFormat)
	ff.codecCtx.time_base.num	= C.int(enc.fpsDen)
	ff.codecCtx.time_base.den	= C.int(enc.fpsNum)
	ff.codecCtx.ticks_per_frame	= 2;
	ff.codecCtx.gop_size		= C.int(enc.gopSize)
	ff.codecCtx.bit_rate		= C.int64_t(enc.Bitrate)

	if C.avcodec_open2(ff.codecCtx, ff.codec, nil) != 0 {
		err = fmt.Errorf("ffmpeg: encoder: avcodec_open2 failed")
		return
	}

	// TODO read some possibly changed params
	// enc.pixelFormat = PixelFormatFF2AV(ff.codecCtx.sample_fmt)
	// enc.FrameSampleCount = int(ff.codecCtx.frame_size)


	// extradata := C.GoBytes(unsafe.Pointer(ff.codecCtx.extradata), ff.codecCtx.extradata_size)
	// fmt.Println("extradata:\n", hex.Dump(extradata))
	// fmt.Println("extradata_size:", len(extradata))


	// Leave codecData uninitialized until SPS and PPS are received (see in encodeOne())
	switch ff.codecCtx.codec_id {
	// case C.AV_CODEC_ID_H264:
	// 	// if enc.codecData, err = h264parser.NewCodecDataFromAVCDecoderConfRecord(extradata[3:]); err != nil {
	// 		fmt.Println("can't init codecData, err:", err)
	// 		return
	// 	}

	default:
		// TODO
		enc.codecData = h264parser.CodecData{
			// codecId: ff.codecCtx.codec_id,
			// pixelFormat: enc.pixelFormat,
			// width: enc.width,
			// height: enc.height,
			// fpsNum: enc.fpsNum,
			// fpsDen: enc.fpsDen,
			// extradata: extradata,
		}
	}

	return
}

func (enc *VideoEncoder) prepare() (err error) {
	ff := &enc.ff.ff
	if ff.frame == nil {
		if err = enc.Setup(); err != nil {
			return
		}
	}
	return
}

// CodecData returns the video codec data of the encoder
func (enc *VideoEncoder) CodecData() (codec av.VideoCodecData, err error) {
	if err = enc.prepare(); err != nil {
		return
	}
	codec = enc.codecData
	return
}

func (enc *VideoEncoder) encodeOne(frame av.VideoFrameRaw) (gotpkt bool, pkt []byte, err error) {
	if err = enc.prepare(); err != nil {
		return
	}

	ff := &enc.ff.ff
	cpkt := C.AVPacket{}
	cgotpkt := C.int(0)

	VideoFrameAssignToFF(frame, ff.frame)

	// Increase pts and convert in 90k: pts * 90000 / fps
	enc.pts++
	ff.frame.pts = C.int64_t( int(enc.pts) * enc.fpsDen * 90000 / enc.fpsNum)

	cerr := C.avcodec_encode_video2(ff.codecCtx, &cpkt, ff.frame, &cgotpkt)
	if cerr < C.int(0) {
		err = fmt.Errorf("ffmpeg: avcodec_encode_video2 failed: %d", cerr)
		return
	}

	if cgotpkt != 0 {
		gotpkt = true
		pkt = C.GoBytes(unsafe.Pointer(cpkt.data), cpkt.size)

		if debug {
			fmt.Println("encoded frame with pts:", cpkt.pts," dts:", cpkt.dts, "duration:", cpkt.duration, "flags:", cpkt.flags)
		}

		// Initialize codecData from SPS and PPS
		// This is done only once, when the first key frame is encoded
		if !enc.codecDataInitialised {
			if (cpkt.flags & C.AV_PKT_FLAG_KEY) != C.AV_PKT_FLAG_KEY {
				fmt.Println("not a keyframe")
			} else {
				var sps, pps []byte
				nalus, _ := h264parser.SplitNALUs(pkt)

				for _, nalu := range nalus {
					if len(nalu) > 0 {
						naltype := nalu[0] & 0x1f
						switch {
						case naltype == 7:
							sps = nalu
						case naltype == 8:
							pps = nalu
						}
					}
				}

				if len(sps) > 0 && len(pps) > 0 {
					enc.codecData, err = h264parser.NewCodecDataFromSPSAndPPS(sps, pps)
					if err != nil {
						fmt.Println("can't init codecData, err:", err)
						return
					}
					enc.codecDataInitialised = true
				} else {
					err = fmt.Errorf("h264parser: empty sps and/or pps")
					fmt.Println("can't init codecData, err:", err)
					return
				}
			}
		}

		C.av_packet_unref(&cpkt)
	} else {
		fmt.Println("ffmpeg: no pkt !")
	}

	return
}


func (enc *VideoEncoder) Encode(frame av.VideoFrameRaw) (pkts [][]byte, err error) {
	var gotpkt bool
	var pkt []byte

	// TODO add converter/scaler
	// if frame.PixelFormat != enc.pixelFormat, width, height etc  {
	// 	if frame, err = enc.resample(frame); err != nil {
	// 		return
	// 	}
	// }

	if gotpkt, pkt, err = enc.encodeOne(frame); err != nil {
		return
	}
	if gotpkt {
		pkts = append(pkts, pkt)
	}

	return
}

func (enc *VideoEncoder) Close() {
	freeFFCtx(enc.ff)
}


func PixelFormatAV2FF(pixelFormat av.PixelFormat) (ffpixelfmt int32) {
	switch pixelFormat {
	case av.I420:
		ffpixelfmt = C.AV_PIX_FMT_YUV420P
	case av.NV12:
		ffpixelfmt = C.AV_PIX_FMT_NV12
	case av.NV21:
		ffpixelfmt = C.AV_PIX_FMT_NV21
	case av.UYVY:
		ffpixelfmt = C.AV_PIX_FMT_UYVY422
	case av.YUYV:
		ffpixelfmt = C.AV_PIX_FMT_YUYV422
	}
	return
}

func PixelFormatFF2AV(ffpixelfmt int32) (pixelFormat av.PixelFormat) {
	switch ffpixelfmt {
	case C.AV_PIX_FMT_YUV420P:
		pixelFormat = av.I420
	case C.AV_PIX_FMT_NV12:
		pixelFormat = av.NV12
	case C.AV_PIX_FMT_NV21:
		pixelFormat = av.NV21
	case C.AV_PIX_FMT_UYVY422:
		pixelFormat = av.UYVY
	case C.AV_PIX_FMT_YUYV422:
		pixelFormat = av.YUYV
	}
	return
}

func (enc *VideoEncoder) SetPixelFormat(fmt av.PixelFormat) (err error) {
	enc.pixelFormat = fmt
	return
}

func (enc *VideoEncoder) SetFramerate(num, den int) (err error) {
	enc.fpsNum = num
	enc.fpsDen = den
	return
}

func (enc *VideoEncoder) SetGopSize(gopSize int) (err error) {
	enc.gopSize = gopSize
	return
}

func (enc *VideoEncoder) SetResolution(w, h int) (err error) {
	enc.width = w
	enc.height = h
	return
}

func (enc *VideoEncoder) SetBitrate(bitrate int) (err error) {
	enc.Bitrate = bitrate
	return
}

func (enc *VideoEncoder) SetOption(key string, val interface{}) (err error) {
	ff := &enc.ff.ff

	sval := fmt.Sprint(val)
	if key == "profile" {
		ff.profile = C.avcodec_profile_name_to_int(ff.codec, C.CString(sval))
		if ff.profile == C.FF_PROFILE_UNKNOWN {
			err = fmt.Errorf("ffmpeg: profile `%s` invalid", sval)
			return
		}
		return
	}

	C.av_dict_set(&ff.options, C.CString(key), C.CString(sval), 0)
	return
}

func (enc *VideoEncoder) GetOption(key string, val interface{}) (err error) {
	ff := &enc.ff.ff
	entry := C.av_dict_get(ff.options, C.CString(key), nil, 0)
	if entry == nil {
		err = fmt.Errorf("ffmpeg: GetOption failed: `%s` not exists", key)
		return
	}
	switch p := val.(type) {
	case *string:
		*p = C.GoString(entry.value)
	case *int:
		fmt.Sscanf(C.GoString(entry.value), "%d", p)
	default:
		err = fmt.Errorf("ffmpeg: GetOption failed: val must be *string or *int receiver")
		return
	}
	return
}


func VideoFrameAssignToAVParams(f *C.AVFrame, frame *av.VideoFrameRaw) {
	frame.PixelFormat = PixelFormatFF2AV(int32(f.format))
	frame.Image.SubsampleRatio = image.YCbCrSubsampleRatio420 // TODO from f.format (AVPixelFormat)
	frame.Image.YStride = int(f.linesize[0])
	frame.Image.CStride = int(f.linesize[1])
	frame.Image.Rect = image.Rectangle{ image.Point{0,0}, image.Point{int(f.width), int(f.height)}}
}

func VideoFrameAssignToAVData(f *C.AVFrame, frame *av.VideoFrameRaw) {
	frame.Image.Y  = C.GoBytes(unsafe.Pointer(f.data[0]), f.linesize[0]*f.height)
	 // TODO chroma subsampling
	frame.Image.Cb = C.GoBytes(unsafe.Pointer(f.data[1]), f.linesize[1]*f.height/2)
	frame.Image.Cr = C.GoBytes(unsafe.Pointer(f.data[2]), f.linesize[2]*f.height/2)
}

func VideoFrameAssignToAV(f *C.AVFrame, frame *av.VideoFrameRaw) {
	VideoFrameAssignToAVParams(f, frame)
	VideoFrameAssignToAVData(f, frame)
}

func VideoFrameAssignToFFParams(frame av.VideoFrameRaw, f *C.AVFrame) {
	// All the following params are described in ffmpeg: frame.h, in struct AVFrame
	f.format = C.int(PixelFormatAV2FF(frame.PixelFormat))
	f.width = C.int(frame.Width())
	f.height = C.int(frame.Height())
	f.sample_aspect_ratio.num = 0
	f.sample_aspect_ratio.den = 1
}

func VideoFrameAssignToFFData(frame av.VideoFrameRaw, f *C.AVFrame) {
	f.data[0] = (*C.uchar)(unsafe.Pointer(&frame.Image.Y[0]))
	f.data[1] = (*C.uchar)(unsafe.Pointer(&frame.Image.Cb[0]))
	f.data[2] = (*C.uchar)(unsafe.Pointer(&frame.Image.Cr[0]))

	f.linesize[0] = C.int(frame.Image.YStride)
	f.linesize[1] = C.int(frame.Image.CStride)
	f.linesize[2] = C.int(frame.Image.CStride)
}

func VideoFrameAssignToFF(frame av.VideoFrameRaw, f *C.AVFrame) {
	VideoFrameAssignToFFParams(frame, f)
	VideoFrameAssignToFFData(frame, f)
}


func NewVideoEncoderByCodecType(typ av.CodecType) (enc *VideoEncoder, err error) {
	var id uint32

	switch typ {
	case av.H264:
		id = C.AV_CODEC_ID_H264

	default:
		fmt.Println("ffmpeg: cannot find encoder codecType=", typ)
		return
	}

	codec := C.avcodec_find_encoder(id)
	if codec == nil || C.avcodec_get_type(id) != C.AVMEDIA_TYPE_VIDEO {
		fmt.Println("ffmpeg: cannot find video encoder codecId=", id)
		return
	}

	_enc := &VideoEncoder{}
	if _enc.ff, err = newFFCtxByCodec(codec); err != nil {
		fmt.Println("could not instantiate enc. err = ", err)
		return
	}
	enc = _enc

	fmt.Println("found enc")
	return
}

func NewVideoEncoderByName(name string) (enc *VideoEncoder, err error) {
	_enc := &VideoEncoder{}

	codec := C.avcodec_find_encoder_by_name(C.CString(name))
	if codec == nil || C.avcodec_get_type(codec.id) != C.AVMEDIA_TYPE_VIDEO {
		err = fmt.Errorf("ffmpeg: cannot find video encoder name=%s", name)
		return
	}

	if _enc.ff, err = newFFCtxByCodec(codec); err != nil {
		return
	}
	enc = _enc
	return
}


// TODO
// func VideoCodecHandler(h *avutil.RegisterHandler) {
// 	h.VideoDecoder = func(codec av.VideoCodecData) (av.VideoDecoder, error) {
// 		if dec, err := NewVideoDecoder(codec); err != nil {
// 			return nil, nil
// 		} else {
// 			return dec, err
// 		}
// 	}

// 	h.VideoEncoder = func(typ av.CodecType) (av.VideoEncoder, error) {
// 		if enc, err := NewVideoEncoderByCodecType(typ); err != nil {
// 			return nil, nil
// 		} else {
// 			return enc, err
// 		}
// 	}
// }




type VideoDecoder struct {
	ff *ffctx
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
	frame *C.AVFrame
}

func (self *VideoFrame) Free() {
	self.Image = image.YCbCr{}
	C.av_frame_free(&self.frame)
}

func freeVideoFrame(self *VideoFrame) {
	self.Free()
}

func (self *VideoDecoder) Decode(pkt []byte) (img av.VideoFrameRaw, err error) {
	ff := &self.ff.ff

	cgotimg := C.int(0)
	frame := C.av_frame_alloc()
	cerr := C.wrap_avcodec_decode_video2(ff.codecCtx, frame, unsafe.Pointer(&pkt[0]), C.int(len(pkt)), &cgotimg)
	if cerr < C.int(0) {
		err = fmt.Errorf("ffmpeg: avcodec_decode_video2 failed: %d", cerr)
		return
	}

	if cgotimg != C.int(0) {
		w := int(frame.width)
		h := int(frame.height)
		ys := int(frame.linesize[0])
		cs := int(frame.linesize[1])

		ffimg := &VideoFrame{Image: image.YCbCr{
			Y: fromCPtr(unsafe.Pointer(frame.data[0]), ys*h),
			Cb: fromCPtr(unsafe.Pointer(frame.data[1]), cs*h/2), // TODO
			Cr: fromCPtr(unsafe.Pointer(frame.data[2]), cs*h/2), // TODO
			YStride: ys,
			CStride: cs,
			SubsampleRatio: image.YCbCrSubsampleRatio420, // TODO
			Rect: image.Rect(0, 0, w, h),
		}, frame: frame}
		runtime.SetFinalizer(ffimg, freeVideoFrame)

		VideoFrameAssignToAV(ffimg.frame, &img)

	} else {
		err = fmt.Errorf("ffmpeg: avcodec_decode_video2 returned no frame")
	}

	return
}

func (enc *VideoDecoder) Close() {
	freeFFCtx(enc.ff)
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
	if err =  _dec.Setup(); err != nil {
		return
	}

	dec = _dec
	return
}

