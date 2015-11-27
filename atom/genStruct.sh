#!/bin/bash

node --harmony_rest_parameters genStruct.js > struct.go && gofmt -w struct.go && go build .

