package main

import (
	"context"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"time"

	pb "github.com/sfproductlabs/scrp/src/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	clusterEndpoint = "backend.local:50551"
)

//Rescrape : Go through load balancer, 're-scrape'
func Rescrape(in *pb.ScrapeRequest) {
	//Update endpoint if we have a cluster address
	if temp := os.Getenv("CLUSTER_ENDPOINT"); temp != "" {
		clusterEndpoint = temp
	}

	// Read cert and key file
	bc := "./.setup/keys/backend.cert"
	if temp := os.Getenv("BACKEND_CERT"); temp != "" {
		bc = temp
	}
	BackendCert, _ := ioutil.ReadFile(bc)

	// Create CertPool
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(BackendCert)

	// Create credentials
	credsClient := credentials.NewClientTLSFromCert(roots, "")

	// Dial with specific Transport (with credentials)
	conn, err := grpc.Dial(clusterEndpoint, grpc.WithTransportCredentials(credsClient))
	if err != nil {
		log.Fatalf("did not connect: %v\n", err)
	}

	defer conn.Close()
	client := pb.NewScraperClient(conn)

	// Contact the server and print out its response.

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.Scrape(ctx, in)
	if err != nil {
		log.Fatalf("could not scrape: %v\n", err)
	}
}
