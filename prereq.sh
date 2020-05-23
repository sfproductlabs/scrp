#!/bin/bash
if [ $# -eq 0 ]
  then
    echo "
            ----------------------------------------------------------
            -- You need GO and CASSANDRA installed to proceed.
            -- Use docker-compose up to start with an empty cassandra
            ----------------------------------------------------------
    "
fi
sleep 5
sudo apt-get install golang autoconf automake libtool curl make g++ unzip #!!! THIS will work on debian/ubuntu
git clone https://github.com/google/protobuf
cd protobuf
./autogen.sh
./configure
make
make check
sudo make install
sudo ldconfig 
go get -u github.com/golang/protobuf/protoc-gen-go
go get -u google.golang.org/grpc
go get -u github.com/sfproductlabs/scrp/src/proto
cd ..

#Setup Cassandra Schema
echo "[CASSANDRA REQUIRED - OR YOU NEED TO INSTALL SCHEMA MANUALLY]"
cqlsh --ssl -f ./.setup/schema.1.cql 

#Generate certificates for gRPC
#Common Name (e.g. server FQDN or YOUR name) []:backend.local
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./.setup/keys/backend.key -out ./.setup/keys/backend.cert -subj "/C=US/ST=San Francisco/L=San Francisco/O=SFPL/OU=IT Department/CN=backend.local"
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./.setup/keys/scrp_scrp.key -out ./.setup/keys/scrp_scrp.cert -subj "/C=US/ST=San Francisco/L=San Francisco/O=SFPL/OU=IT Department/CN=scrp_scrp"
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./.setup/keys/frontend.key -out ./.setup/keys/frontend.cert -subj "/C=US/ST=San Francisco/L=San Francisco/O=SFPL/OU=IT Department/CN=frontend.local"


#go generate github.com/dioptre/scrp/src/proto
cd src
protoc -I proto/ proto/helloworld.proto proto/scrape.proto --go_out=plugins=grpc:proto/
#PATH=/app/scrp/go/bin:/app/golang/bin:$PATH protoc -I proto/ proto/helloworld.proto proto/scrape.proto --go_out=plugins=grpc:proto/

cd ..
