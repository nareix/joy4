
avconv -loglevel debug -y -i /tmp/tiny2.mov \
	-bsf h264_mp4toannexb -an -vcodec copy -strict experimental /tmp/out.ts 2>&1

