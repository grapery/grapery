#!/bin/bash

go env -w GO111MODULE=on
export PATH="/home/ubuntu/work/bin":$PATH

IncludePath=/snap/protobuf/current/include
GPATH=/home/ubuntu/work
basepath=$PWD
project_dir=github.com/grapery/grapery
pb_dir=common-protoc
proto_install="$PROTOC_INSTALL"
go_package=api
rm -rf $go_package

for i in $(ls $basepath/$pb_dir/*.proto); do
	fn=$pb_dir/$(basename "$i")
	echo $fn
	protoc -I$IncludePath -I. \
		-I$GPATH/src \
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--go_out=plugins=grpc:. "$fn"
	protoc -I$IncludePath -I. \
		-I$GPATH/src \
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--grpc-gateway_out=logtostderr=true:. "$fn"
	protoc -I$IncludePath -I. \
		-I$GPATH/src \
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--openapiv2_out . \
        --openapiv2_opt logtostderr=true \
        --openapiv2_opt use_go_templates=true \
		"$fn" 
done
