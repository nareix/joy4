package common

type TNALUInfos struct {
	NALUFormat string
	Infos      []TNALUInfo
}

type TNALUInfo struct {
	UnitType  string
	RefIdc    int
	NumBytes  int
	Data      []byte
	SliceType string
}
