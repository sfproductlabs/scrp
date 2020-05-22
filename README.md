# Scrp
A fully resumable horizontally (infinitely) scalable webscraper in Go. Think 1000's of machines scraping sites in a distributed way. Based on Docker Swarm, Cassandra, colly, gRPC, and my other [boilerplate](https://github.com/dioptre/gtrpc).

## Note
Why are you even here? Maybe you could probably just use colly... Especially if you don't care about scalability... or use a shell script ([Example](https://github.com/sfproductlabs/scrp/blob/master/simple.sh))

## Why
I built this to distribute scraping across multiple servers, so as to go undetected. I could have used proxies, but wanted to reuse the code for other distributed apps.

## Local Execution

### Installing (local)
Run:
```
docker-compose up
```
Then (on linux - you can use brew on mac):
```
#apt install go
#./prereq.sh
#./build.sh
```

### Scrape Instructions (local)
First run the server on all the nodes (use the testlocal.sh script for brevity):
```
#GOCQL_HOST_LOOKUP_PREFER_V4=true /usr/bin/nice -n 5 ./gsvc localhost false false ./.setup/keys/rootCa.crt ./.setup/keys/cassandra-client.crt ./.setup/keys/cassandra-client.key
```
Notice the parameters:
```
[0] - cassandra-databases (comma-separated, no spaces)
[1] - cassandra-retry (should we retry execution on the cassandra cluster)
[2] - cassandra-veify (should we verify the cassandra service)
[3] - cassandra-rootca  (only use this if you need)
[4] - cassandra-client-cert (only use this if you need)
[5] - cassandra-client-key (only use this if you need)
```

Then send a request via the client:
```
./gcli https://en.wikipedia.org/wiki/List_of_HTTP_status_codes
```

## Running on Docker Swarm
(TODO)