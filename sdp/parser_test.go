package sdp

import (
	"testing"
)

func TestParse(t *testing.T) {
	infos := Decode(`
v=0
o=- 1459325504777324 1 IN IP4 192.168.0.123
s=RTSP/RTP stream from Network Video Server
i=mpeg4cif
t=0 0
a=tool:LIVE555 Streaming Media v2009.09.28
a=type:broadcast
a=control:*
a=range:npt=0-
a=x-qt-text-nam:RTSP/RTP stream from Network Video Server
a=x-qt-text-inf:mpeg4cif
m=video 0 RTP/AVP 96
c=IN IP4 0.0.0.0
b=AS:300
a=rtpmap:96 H264/90000
a=fmtp:96 packetization-mode=1;profile-level-id=640028;sprop-parameter-sets=Z2QAKK2EBUViuKxUdCAqKxXFYqOhAVFYrisVHQgKisVxWKjoQFRWK4rFR0ICorFcVio6ECSFITk8nyfk/k/J8nm5s00IEkKQnJ5Pk/J/J+T5PNzZprQFoe0qQAAAHgAABDgYEABJPAAUmW974XhEI1A=,aO48sA==;config=0000000167640028ad84054562b8ac5474202a2b15c562a3a1015158ae2b151d080a8ac57158a8e84054562b8ac5474202a2b15c562a3a10248521393c9f27e4fe4fc9f279b9b34d081242909c9e4f93f27f27e4f93cdcd9a6b405a1ed2a4000001e00000438181000493c0014996f7be1784423500000000168ee3cb0
a=x-dimensions: 720, 480
a=x-framerate: 15
a=control:track1
m=audio 0 RTP/AVP 96
c=IN IP4 0.0.0.0
b=AS:256
a=rtpmap:96 MPEG4-GENERIC/16000/2
a=fmtp:96 streamtype=5;profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3;config=1408
a=control:track2
`)
	t.Logf("%v", infos)
}
