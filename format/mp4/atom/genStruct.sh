#!/bin/bash

node genStruct.js > struct.go && gofmt -w struct.go && go build . || {
	echo
	echo "Please use node version > 6.0.0"
}

