#!/bin/bash

#go generate github.com/dioptre/scrp/src/proto
cd src; protoc -I proto/ proto/helloworld.proto proto/scrape.proto --go_out=plugins=grpc:proto/; cd ..;

#Update version
oldver=`grep -oP '(?<=Scrp\.\ Version\ ).*(?=\")' ./src/service/service.go`
newver=`expr $oldver + 1`
sed -i "s/Scrp\.\ Version\ $oldver/Scrp\.\ Version\ $newver/g" "./src/service/service.go"

#Build client & server
go build -o gsvc -tags netgo src/service/*.go
go build -o gcli -tags netgo src/client/*.go

#go install github.com/sfproductlabs/scrp
sudo docker build -t scrp .
