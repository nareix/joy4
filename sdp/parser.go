package sdp

import (
	"strings"
	"fmt"
	"strconv"
	"encoding/hex"
	"encoding/base64"
	"github.com/nareix/av"
)

type Info struct {
	AVType string
	Type int
	TimeScale int
	Control string
	Rtpmap int
	Config []byte
	SpropParameterSets [][]byte
	PayloadType int
	SizeLength int
	IndexLength int
}

func Decode(content string) (infos []Info) {
	var info *Info

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		typeval := strings.SplitN(line, "=", 2)
		if len(typeval) == 2 {
			fields := strings.SplitN(typeval[1], " ", 2)

			switch typeval[0] {
			case "m":
				if len(fields) > 0 {
					switch fields[0] {
					case "audio", "video":
						infos = append(infos, Info{AVType: fields[0]})
						info = &infos[len(infos)-1]
						mfields := strings.Split(fields[1], " ")
						if len(mfields) >= 3 {
							info.PayloadType, _ = strconv.Atoi(mfields[2])
						}
					}
				}

			case "a":
				if info != nil {
					for _, field := range fields {
						keyval := strings.SplitN(field, ":", 2)
						if len(keyval) >= 2 {
							key := keyval[0]
							val := keyval[1]
							switch key {
							case "control":
								info.Control = val
							case "rtpmap":
								info.Rtpmap, _ = strconv.Atoi(val)
							}
						}
						keyval = strings.Split(field, "/")
						if len(keyval) >= 2 {
							key := keyval[0]
							switch key {
							case "MPEG4-GENERIC":
								info.Type = av.AAC
							case "H264":
								info.Type = av.H264
							}
							if i, err := strconv.Atoi(keyval[1]); err == nil {
								info.TimeScale = i
							}
							if false {
								fmt.Println("sdp:", keyval[1], info.TimeScale)
							}
						}
						keyval = strings.Split(field, ";")
						if len(keyval) > 1 {
							for _, field := range keyval {
								keyval := strings.SplitN(field, "=", 2)
								if len(keyval) == 2 {
									key := strings.TrimSpace(keyval[0])
									val := keyval[1]
									switch key {
									case "config":
										info.Config, _ = hex.DecodeString(val)
									case "sizelength":
										info.SizeLength, _ = strconv.Atoi(val)
									case "indexlength":
										info.IndexLength, _ = strconv.Atoi(val)
									case "sprop-parameter-sets":
										fields := strings.Split(val, ",")
										for _, field := range fields {
											val, _ := base64.StdEncoding.DecodeString(field)
											info.SpropParameterSets = append(info.SpropParameterSets, val)
										}
									}
								}
							}
						}
					}
				}

			}
		}
	}
	return
}

