#!/bin/bash

go env -w GO111MODULE=on

if [ -z "$PROTOC_INSTALL" ]; then
	echo "PROTOC_INSTALL not set"
	exit 1
fi

GOPATH=/home/ubuntu/work
basepath=$GOPATH/src
pb_package=github.com/grapery/grapery/pb
proto_install="$PROTOC_INSTALL"
go_package=api
rm -rf $go_package

for i in $(ls $basepath/$pb_package/*.proto); do
	echo $i
	fn=$pb_package/$(basename "$i")
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--go_out=plugins=grpc:. "$fn"
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--grpc-gateway_out=logtostderr=true:. "$fn"
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--swagger_out=logtostderr=true:. "$fn"
done
