
package atom

func GetAVCDecoderConfRecordByTrack(track *Track) (record *AVCDecoderConfRecord) {
	if media := track.Media; media != nil {
		if info := media.Info; info != nil {
			if sample := info.Sample; sample != nil {
				if desc := sample.SampleDesc; desc != nil {
					if avc1 := desc.Avc1Desc; avc1 != nil {
						if conf := avc1.Conf; conf != nil {
							return &conf.Record
						}
					}
				}
			}
		}
	}
	return
}

func GetMp4aDescByTrack(track *Track) (mp4a *Mp4aDesc) {
	if media := track.Media; media != nil {
		if info := media.Info; info != nil {
			if sample := info.Sample; sample != nil {
				if desc := sample.SampleDesc; desc != nil {
					if mp4a = desc.Mp4aDesc; mp4a != nil {
						return
					}
				}
			}
		}
	}
	return
}

