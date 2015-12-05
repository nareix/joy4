
./avconv -loglevel debug -y -i /tmp/tiny2.mov \
	-bsf h264_mp4toannexb -acodec copy -vcodec copy -strict experimental /tmp/out.ts 2>&1

