package sdp

import (
	"strings"
	"strconv"
	"encoding/hex"
	"encoding/base64"
	"github.com/nareix/av"
)

type Info struct {
	AVType string
	Type av.CodecType
	TimeScale int
	Control string
	Rtpmap int
	Config []byte
	SpropParameterSets [][]byte
}

func Decode(content string) (infos []Info) {
	var info *Info

	for _, line := range strings.Split(content, "\n") {
		line = strings.Trim(line, "\r")
		typeval := strings.SplitN(line, "=", 2)
		if len(typeval) == 2 {
			fields := strings.Split(typeval[1], " ")

			switch typeval[0] {
			case "m":
				if len(fields) > 0 {
					switch fields[0] {
					case "audio", "video":
						infos = append(infos, Info{AVType: fields[0]})
						info = &infos[len(infos)-1]
					}
				}

			case "a":
				if info != nil {
					for _, field := range fields {
						keyval := strings.Split(field, ":")
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
								info.TimeScale, _ = strconv.Atoi(keyval[1])
							case "H264":
								info.Type = av.H264
								info.TimeScale, _ = strconv.Atoi(keyval[1])
							}
						}
						keyval = strings.Split(field, ";")
						if len(keyval) > 1 {
							for _, field := range keyval {
								keyval := strings.SplitN(field, "=", 2)
								if len(keyval) == 2 {
									key := keyval[0]
									val := keyval[1]
									switch key {
									case "config":
										info.Config, _ = hex.DecodeString(val)
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

