
codec
====

Golang aac/h264 encoder and decoder

H264 encoding example

	w := 400
	h := 400
	var nal [][]byte 

	c, _ := codec.NewH264Encoder(w, h, image.YCbCrSubsampleRatio420)
	nal = append(nal, c.Header)

	for i := 0; i < 60; i++ {
		img := image.NewYCbCr(image.Rect(0,0,w,h), image.YCbCrSubsampleRatio420)
		p, _ := c.Encode(img)
		if len(p.Data) > 0 {
			nal = append(nal, p.Data)
		}
	}
	for {
		// flush encoder
		p, err := c.Encode(nil)
		if err != nil {
			break
		}
		nal = append(nal, p.Data)
	}

H264 decoding example

	dec, err := codec.NewH264Decoder(nal[0])
	for i, n := range nal[1:] {
		img, err := dec.Decode(n)
		if err == nil {
			fp, _ := os.Create(fmt.Sprintf("/tmp/dec-%d.jpg", i))
			jpeg.Encode(fp, img, nil)
			fp.Close()
		}
	}

AAC encoding example
	
	var pkts [][]byte 

	c, _ := codec.NewAACEncoder()
	pkts = append(pkts, c.Header)

	for i := 0; i < 60; i++ {
		var sample [8192]byte
		p, _ := c.Encode(sample)
		if len(p) > 0 {
			pkts = append(pkts, p)
		}
	}

AAC decoding example
	
	dec, _ := codec.NewAACDecoder(pkts[0])
	for _, p := range pkts[1:] {
		sample, err := dec.Decode(p)
	}

