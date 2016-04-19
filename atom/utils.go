package atom

func GetAvc1ConfByTrack(stream *Track) (avc1 *Avc1Conf) {
	if media := stream.Media; media != nil {
		if info := media.Info; info != nil {
			if sample := info.Sample; sample != nil {
				if desc := sample.SampleDesc; desc != nil {
					if avc1 := desc.Avc1Desc; avc1 != nil {
						return avc1.Conf
					}
				}
			}
		}
	}
	return
}

func GetMp4aDescByTrack(stream *Track) (mp4a *Mp4aDesc) {
	if media := stream.Media; media != nil {
		if info := media.Info; info != nil {
			if sample := info.Sample; sample != nil {
				if desc := sample.SampleDesc; desc != nil {
					return desc.Mp4aDesc
				}
			}
		}
	}
	return
}
