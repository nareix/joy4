go-av
==

Golang audio and video manipulation library. including mp4, rtmp, yuv/rgb image converter

H264,AAC Codec
====

Requires `libav` installed.

    d, err = codec.NewAACEncoder()
    data, err = d.Encode(samples)
    
    d, err = codec.NewAACDecoder(aaccfg)
    samples, err = d.Decode(data)
    
    var img *image.YCbCr
    d, err = codec.NewH264Encoder(640, 480)
    img, err = d.Encode(img)
    
    d, err = codec.NewH264Decoder(pps)
    img, err = d.Decode(nal)
