package ffmpeg

/*
#include "ffmpeg.h"
int wrap_avcodec_decode_video2(AVCodecContext *ctx, AVFrame *frame, void *data, int size, int *got) {
	struct AVPacket pkt = {.data = data, .size = size};
	return avcodec_decode_video2(ctx, frame, got, &pkt);
}
int wrap_av_image_alloc(uint8_t *pointers[4], int linesizes[4], int w, int h, enum AVPixelFormat pix_fmt, int align) {
	return av_image_alloc(pointers, linesizes, w, h, pix_fmt, align);
}
*/
import "C"
import (
	"unsafe"
	"fmt"
	"reflect"
	"image"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
)


type VideoScaler struct {
	inPixelFormat, OutPixelFormat av.PixelFormat
	inWidth, OutWidth int
	inHeight, OutHeight int
	inYStride, OutYStride int
	inCStride, OutCStride int
	swsCtx *C.struct_SwsContext
}

func (self *VideoScaler) Close() {
	C.sws_freeContext(self.swsCtx);
}


func (self *VideoScaler) AllocOutputImage(strides (*[3]C.int)) (dataPtr ([4]*C.uint8_t), bufSize int, err error) {
	align := 16 // align buffer size on 16 pixels for the encoder

	// The allocated image buffer has to be freed by using av_freep(&pointers[0]).
	bufSize = int(C.wrap_av_image_alloc(&dataPtr[0], &strides[0], C.int(self.OutWidth), C.int(self.OutHeight), PixelFormatAV2FF(self.OutPixelFormat), C.int(align)))
	if bufSize < 0 {
		err = fmt.Errorf("Could not allocate image\n");
	}
	return
}

func (self *VideoScaler) videoScaleOne(src av.VideoFrameRaw) (dst av.VideoFrameRaw, err error) {
	var srcPtr ([3]*C.uint8_t)
	srcPtr[0] = (*C.uint8_t)(src.Y)
	srcPtr[1] = (*C.uint8_t)(src.Cb)
	srcPtr[2] = (*C.uint8_t)(src.Cr)

	var inStrides ([3]C.int)
	inStrides[0] = C.int(src.YStride)
	inStrides[1] = C.int(src.CStride)
	inStrides[2] = C.int(src.CStride)

	var outStrides ([3]C.int)
	outStrides[0] = C.int(self.OutYStride)
	outStrides[1] = C.int(self.OutCStride)
	outStrides[2] = C.int(self.OutCStride)
	
	dstPtr, _, err := self.AllocOutputImage(&outStrides)
	if err != nil {
		return
	}


	// convert to destination format and resolution
	C.sws_scale(self.swsCtx, &srcPtr[0], &inStrides[0], 0, C.int(self.inHeight), &dstPtr[0], &outStrides[0])


	dst.PixelFormat	= PixelFormatFF2AV(int32(self.OutPixelFormat))
	dst.YStride		= int(outStrides[0])
	dst.CStride		= int(outStrides[1])
	// TODO dst.SubsampleRatio =
	dst.Rect		= image.Rect(0, 0, self.OutWidth, self.OutHeight)
	dst.Y			= unsafe.Pointer(dstPtr[0])
	dst.Cb			= unsafe.Pointer(dstPtr[1])
	dst.Cr			= unsafe.Pointer(dstPtr[2])


	// C.memset(dst.Y,  128, C.ulong(dst.YStride * self.OutHeight))
	// C.memset(dst.Cb, 128, C.ulong(dst.CStride * self.OutHeight/2))
	// C.memset(dst.Cr, 128, C.ulong(dst.CStride * self.OutHeight/2))

	// fmt.Println("dst.YStride * self.OutHeight:", dst.YStride * self.OutHeight)
	// fmt.Println("dst.CStride * self.OutHeight/2:", dst.CStride * self.OutHeight/2)
	// dst.Dump("framescale.yuv")


	// fmt.Println("Scaling succeeded: pix_fmt:", self.OutPixelFormat, "resolution:", self.OutWidth, self.OutHeight)
	// C.av_freep(&dstPtr[0]) // TODO callback to free
	return
}


func (self *VideoScaler) VideoScale(src av.VideoFrameRaw) (dst av.VideoFrameRaw, err error) {
	if self.swsCtx == nil {
		self.inPixelFormat = src.PixelFormat
		self.inWidth = src.Width()
		self.inHeight= src.Height()
		self.inYStride = src.YStride
		self.inCStride = src.CStride

		self.swsCtx = C.sws_getContext(C.int(self.inWidth), C.int(self.inHeight), PixelFormatAV2FF(self.inPixelFormat),
			C.int(self.OutWidth), C.int(self.OutHeight), PixelFormatAV2FF(self.OutPixelFormat),
			C.SWS_BILINEAR, (*C.SwsFilter)(C.NULL), (*C.SwsFilter)(C.NULL), (*C.double)(C.NULL))

		if self.swsCtx == nil {
			err = fmt.Errorf("Impossible to create scale context for the conversion fmt:%d s:%dx%d -> fmt:%d s:%dx%d\n",
				/*C.av_get_pix_fmt_name*/(self.inPixelFormat), self.inWidth, self.inHeight,
				/*C.av_get_pix_fmt_name*/(self.OutPixelFormat), self.OutWidth, self.OutHeight);
			return
		}

		fmt.Println("VideoScaler:\n", self)
	}

	dst, err = self.videoScaleOne(src)
	return
}


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
	scaler *VideoScaler
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

	// VideoFrameAssignToFF(frame, ff.frame)
	ff.frame.format = C.int32_t(PixelFormatAV2FF(frame.GetPixelFormat()))

	ys, cs := frame.GetStride()
	ff.frame.linesize[0] = C.int(ys)
	ff.frame.linesize[1] = C.int(cs)
	ff.frame.linesize[2] = C.int(cs)

	w, h := frame.GetResolution()
	ff.frame.width = C.int(w)
	ff.frame.height = C.int(h)
	ff.frame.sample_aspect_ratio.num = 0 // TODO
	ff.frame.sample_aspect_ratio.den = 1

	data0, data1, data2 := frame.GetDataPtr()
	ff.frame.data[0] = (*C.uchar)(data0)
	ff.frame.data[1] = (*C.uchar)(data1)
	ff.frame.data[2] = (*C.uchar)(data2)

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


func (self *VideoEncoder) scale(in av.VideoFrameRaw) (out av.VideoFrameRaw, err error) {
	if self.scaler == nil {
		self.scaler = &VideoScaler{
			inPixelFormat:	in.GetPixelFormat(),
			inWidth:		in.Width(),
			inHeight:		in.Height(),
			inYStride:		in.YStride,
			inCStride:		in.CStride,
			OutPixelFormat: in.GetPixelFormat(), // TODO
			OutWidth:		self.width,
			OutHeight:		self.height,
			OutYStride:		self.width, // TODO
			OutCStride:		self.width/2, // TODO
		}
	}
	if out, err = self.scaler.VideoScale(in); err != nil {
		return
	}
	return
}


func (enc *VideoEncoder) Encode(frame av.VideoFrameRaw) (pkts [][]byte, err error) {
	var gotpkt bool
	var pkt []byte

	if frame.PixelFormat != enc.pixelFormat || frame.Width() != enc.width || frame.Height() != enc.height/* TODO add stride ? */ {
		if frame, err = enc.scale(frame); err != nil {
			return nil, err
		}
	}

	if gotpkt, pkt, err = enc.encodeOne(frame); err != nil {
		return nil, err
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

		// SubsampleRatio: image.YCbCrSubsampleRatio420, // TODO

		// VideoFrameAssignToAV(ffimg.frame, &img)
		img.SetPixelFormat(PixelFormatFF2AV(int32(C.AV_PIX_FMT_YUV420P)))
		img.SetStride(ys, cs)
		img.SetResolution(w, h)
		img.SetDataPtr( unsafe.Pointer(frame.data[0]), unsafe.Pointer(frame.data[1]), unsafe.Pointer(frame.data[2]))

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

