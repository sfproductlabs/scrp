/*
 *
 * Copyright 2015 gRPC authors.
 * & Andrew Grosser
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

//TODO: AG
//Use GO_REUSEPORT listener
//Run a separate server instance per CPU core with GOMAXPROCS=1 (it appears during benchmarks that there is a lot more context switches with Traefik than with nginx)

//Example
//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto
package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	random "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/gocql/gocql"
	pb "github.com/sfproductlabs/scrp/src/proto"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	port         = ":50551"
	internal     = false //run requests on the same server or go through load balancer
	debugScraper = true  //print verbose debug output
	processWait  = 14    //seconds random max wait time for query outstanding links
	siteWait     = 14    //seconds to wait between hits on site
	retries      = 1
)

var (
	midOnce sync.Once
	mid     [6]byte
	mids    string
	//Setup cassandra connection and defaults
	db = &Cassandra{URLs: []string{"127.0.0.1"}, Keyspace: "scrp", VerifyHost: true, Retry: false}
)

type server struct{}

func getMachineID() {
	midOnce.Do(func() {
		io.ReadFull(rand.Reader, mid[:])
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, iface := range ifaces {
				if len(iface.HardwareAddr) >= 6 {
					copy(mid[:], iface.HardwareAddr)
					break
				}
			}
		}
		mids = base64.StdEncoding.EncodeToString(mid[:6])
	})
}

//GetMachineBytes : Gets the server's unique ID
func GetMachineBytes() [6]byte {
	getMachineID()
	return mid
}

//GetMachineString : Gets the server's unique ID
func GetMachineString() string {
	getMachineID()
	return mids
}

// SayHello implements proto.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

// Scrape implements proto.ScaperServer
func (s *server) Scrape(ctx context.Context, in *pb.ScrapeRequest) (*pb.ScrapeReply, error) {
	//Notify the receiver of a new URL
	if in.Id == "" {
		in.Id = gocql.TimeUUID().String()
	}
	received <- in
	return &pb.ScrapeReply{Message: in.Id}, nil
}

func scrape(in *pb.ScrapeRequest) {
	//ctx := context.Background()

	// Instantiate default collector
	c := colly.NewCollector(
		colly.Async(true), // Turn on asynchronous requests
		colly.IgnoreRobotsTxt(),
		//Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0
		//Mozilla/5.0 (Macintosh; Intel Mac OS X x.y; rv:42.0) Gecko/20100101 Firefox/42.0
		//"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
		//User-Agent: Mediapartners-Google Or for image search:   User-Agent: Googlebot-Image/1.0
		colly.UserAgent("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"),
	)

	if debugScraper {
		// Attach a debugger to the collector
		colly.Debugger(&debug.LogDebugger{})(c)
	}

	if in.Filter != "" && in.Filter != "_" {
		filterStrings := strings.Split(in.Filter, "||")
		filters := make([]*regexp.Regexp, len(filterStrings))
		for i := range filterStrings {
			filters[i] = regexp.MustCompile(filterStrings[i])
		}
		colly.URLFilters(filters...)(c)
	}

	if in.Domain != "" && in.Domain != "_" {
		colly.AllowedDomains(strings.Split(in.Domain, ",")...)(c)
	}

	// Limit the number of threads started by colly to one
	// when visiting links which domains' matches "*httpbin.*" glob
	c.Limit(&colly.LimitRule{
		Parallelism: 1,
		//Delay:       7 * time.Second,
		RandomDelay: 14 * time.Second,
	})

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		//in.Url = e.Attr("href")
		u, _ := url.Parse(e.Attr("href"))
		newURL := e.Request.URL.ResolveReference(u)
		in.Url = newURL.String()
		approved := false
		if c.AllowedDomains == nil || len(c.AllowedDomains) == 0 {
			approved = true
		} else {
			for _, d := range c.AllowedDomains {
				if d == newURL.Host {
					approved = true
					break
				}
			}
		}
		if approved == true {
			if len(c.URLFilters) > 0 {
				approved = false
				for _, r := range c.URLFilters {
					if r.Match([]byte(in.Url)) {
						approved = true
						break
					}
				}
			}
			if approved == true {
				if internal {
					received <- in
				} else {
					Rescrape(in)
				}
			}

		}

	})

	ScrapeDetail(in, c)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept-Language", "en-US")
		r.Headers.Set("From", "googlebot(at)googlebot.com")
		fmt.Println("Visiting", r.URL)
	})

	//TODO:Save original in S3
	c.OnResponse(func(r *colly.Response) {
		//fmt.Println(string(r.Body))
		in.Status = int32(r.StatusCode)
		in.Size = int64(len(r.Body))
		db.UpdateURL(in)
	})

	c.OnError(func(r *colly.Response, err error) {
		in.Status = int32(r.StatusCode)
		db.UpdateURL(in)
	})

	c.Visit(fmt.Sprintf("%s", in.Url))

	// Wait until threads are finished
	c.Wait()
}

//Let's not receive more than 1000 links at a time (per machine)
var received = make(chan *pb.ScrapeRequest, 1000)

func receive() {
RECEIVE:
	var in = <-received
	//Write as fast as we want to cassandra, and add new ones to the queue
	db.InsertURL(in)
	goto RECEIVE //TODO: Test whether this as a single thread is ok
}

//Query share the common query type
type Query struct {
	Domain string
	Filter string
	Queue  chan *pb.ScrapeRequest
}

var queries = make(map[gocql.UUID]*Query)

//Provide order to the system and limit amount of connections per crawler
//Think about using the leaky bucket, and a worker pool
//https://gobyexample.com/worker-pools
//& a bursty limiter
//https://gobyexample.com/rate-limiting
//TODO: Could also optimize memory allocation of collys with QueryId
//TODO: Write a service to go through unfinished links in the background
////Use len(queued) to see how many we have queued already
//TODO: Check if we've already got it in cassandra - else scrape!!!!
//time.Sleep(1 * time.Second)
// c := Cassandra{}
// c.Description()
//import "net"
//import "net/url"
// fmt.Println(u.Host)
// host, port, _ := net.SplitHostPort(u.Host)
// fmt.Println(host)
// fmt.Println(port)
//TODO: Rate limit the scrape by ScrapeRequest.Id
func process() {
	var url string
	var qid gocql.UUID
	var seq gocql.UUID
	var status int32
	var sched time.Time
	var mid string
	var attempt gocql.UUID
	var attempts int32
	var domain string
	var filter string
	var queue chan *pb.ScrapeRequest
PROCESS:
	iter := db.GetTodos()
	for {
		// New map each iteration
		row := map[string]interface{}{
			"url":      &url,
			"qid":      &qid,
			"seq":      &seq,
			"status":   &status,
			"sched":    &sched,
			"mid":      &mid,
			"attempt":  &attempt,
			"attempts": &attempts,
		}
		if !iter.MapScan(row) {
			break
		}
		if sched.After(time.Now()) {
			continue
		}
		if attempt.Time().In(time.UTC).Add(12 * time.Second).After(time.Now().In(time.UTC)) {
			continue
		}
		if val, ok := queries[qid]; ok {
			domain = val.Domain
			filter = val.Filter
			queue = val.Queue
		} else {
			var err error
			if domain, filter, err = db.GetQuery(qid.String()); err == nil {
				db.UpdateAttempt(url)
				queue = make(chan *pb.ScrapeRequest, 100)
				queries[qid] = &Query{Domain: domain, Filter: filter, Queue: queue}
				go func() {
					for {
						var approved = <-queue
						scrape(approved)
						time.Sleep((time.Duration(random.Intn(siteWait)) + 3) * time.Second)
					}
				}()
			}
		}
		queue <- &pb.ScrapeRequest{Id: qid.String(), Url: url, Domain: domain, Filter: filter, Seq: seq.String(), Status: status, Sched: sched.String(), Mid: mid, Attempts: attempts}
	}
	time.Sleep(time.Duration(random.Intn(processWait)) * time.Second)
	goto PROCESS
}

func main() {
	fmt.Println("\n\n//////////////////////////////////////////////////////////////")
	fmt.Println("Scrp. Version 17")
	fmt.Println("Horizontal web-scraper for clusters and swarm")
	fmt.Println("https://github.com/sfproductlabs/scrp")
	fmt.Println("(c) Copyright 2020 SF Product Labs LLC.")
	fmt.Println("Use of this software is subject to the LICENSE agreement.")
	fmt.Println("//////////////////////////////////////////////////////////////\n\n ")

	//Cassandra Hosts
	if len(os.Args) > 1 {
		db.URLs = strings.Split(os.Args[1], ",")
	}
	//Cassandra Retry
	if len(os.Args) > 2 {
		v, err := strconv.ParseBool(os.Args[2])
		if err != nil {
			v = false
		}
		db.Retry = v
	}
	//Cassandra VerifyHost
	if len(os.Args) > 3 {
		v, err := strconv.ParseBool(os.Args[3])
		if err != nil {
			v = false
		}
		db.VerifyHost = v
	}
	//Cassandra Cert
	if len(os.Args) > 6 {
		db.SSLCA = os.Args[4]
		db.SSLCert = os.Args[5]
		db.SSLKey = os.Args[6]
	}
	//Cassandra Username & Password
	if len(os.Args) > 8 {
		db.Username = os.Args[7]
		db.Password = os.Args[8]
	}

	if err := db.Connect(); err != nil {
		log.Fatalf("Failed to connect to Cassandra:\n\n%v\n\nError:\n%v", db, err)
	}
	fmt.Println("Connected to Cassandra")

	//Let's receive requests
	go receive()

	//Let's process approved requests
	go process()

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "OK")
		})

		http.ListenAndServe(":50580", nil)
	}()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Read cert and key file
	bc := "./.setup/keys/backend.cert"
	if temp := os.Getenv("BACKEND_CERT"); temp != "" {
		bc = temp
	}
	bk := "./.setup/keys/backend.key"
	if temp := os.Getenv("BACKEND_KEY"); temp != "" {
		bk = temp
	}
	BackendCert, _ := ioutil.ReadFile(bc)
	BackendKey, _ := ioutil.ReadFile(bk)

	// Generate Certificate struct
	cert, err := tls.X509KeyPair(BackendCert, BackendKey)
	if err != nil {
		log.Fatalf("Failed to parse backend certificate %s %s: %v", bc, bk, err)
	}

	// Create credentials
	creds := credentials.NewServerTLSFromCert(&cert)

	// Use Credentials in gRPC server options
	serverOption := grpc.Creds(creds)
	var s = grpc.NewServer(serverOption)
	defer s.Stop()

	pb.RegisterScraperServer(s, &server{})
	fmt.Printf("Server up on %s\n", port)
	// Register reflection service on gRPC server.
	// reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
