#!/bin/bash

go env -w GO111MODULE=on

export PATH="/home/ubuntu/work/bin":$PATH
IncludePath=/snap/protobuf/current/include
GPATH=/home/ubuntu/work

# GPATH=$GOPATH
# IncludePath="$PROTOC_INSTALL"

basepath=$PWD

pb_dir=common-protoc
go_package=api
rm -rf $go_package

for i in $(ls $basepath/$pb_dir/*.proto); do
	fn=$pb_dir/$(basename "$i")
	echo $fn
	protoc -I$IncludePath -I. \
		-I$GPATH/src \
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--go_out=. --go_opt=paths=source_relative "$fn"
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
	protoc -I$proto_install/include -I. \
		-I$GOPATH/src \
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/\
		-I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis\
		--validate_out="lang=go:". "$fn"
done

cp common-protoc/*.go api/service
