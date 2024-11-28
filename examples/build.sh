#!/usr/bin/env bash

#go交叉编译
export CGO_ENABLED=0 GOOS=linux GOARCH=amd64

rm -rf goreplay_middle_bin
go build  -o goreplay_middle_bin ./examples