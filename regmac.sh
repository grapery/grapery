#!/bin/bash

if [ -z "$PROTOC_INSTALL" ]; then
	echo "PROTOC_INSTALL not set"
	exit 1
fi

basepath=$GOPATH/src
pb_package=github.com/grapery/grapery/common-protoc
proto_install="$PROTOC_INSTALL"
go_package=api
rm -rf $go_package

cd $basepath
for i in $(ls $basepath/$pb_package/*.proto); do
	echo $i
	fn=$pb_package/$(basename "$i")
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/googleapis/\
		--go_out=. --go-grpc_out=require_unimplemented_servers=false:. "$fn"
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/googleapis/\
		--grpc-gateway_out=. "$fn"
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/googleapis/\
		--openapiv2_out=logtostderr=true:. "$fn"
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/googleapis/\
		--validate_out="lang=go:". "$fn"
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/googleapis/\
		--swift_out=/Users/grapestree/Desktop/apps/voyager3/voyager3/ "$fn"
done
