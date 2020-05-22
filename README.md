# Scrp
A fully resumable horizontally (infinitely) scalable webscraper in Go. Think 1000's of machines scraping sites in a distributed way. Based on Docker Swarm, Cassandra, colly, gRPC, and my other [boilerplate](https://github.com/dioptre/gtrpc).

## Note
Why are you even here? Maybe you could probably just use colly... Especially if you don't care about scalability... or use a shell script, for example:
```bash
#!/bin/bash
url="http://www.cityfeet.com/cont/api/search/listings-spatial"
cookie="ASP.NET_SessionId=x335iekckm5tqxcq12psv1p2; __RequestVerificationToken_L2NvbnQ1=FTTyjLMPpvjTLNYvWo5a5yFqhos830-fpyjtxwr4vsVnG8P7_bf5zEEpH4JjY2KfIKgHMuuotd9IyW4iUmSeYRHnLzQ1"
DATE=`date +%Y-%m-%d`
#for i in $(cat query.txt); do
for i in {1..9}; do
    body="{'location':{'name':'San Francisco, CA','bb':[37.708131,-122.51777,37.863424,-122.3570311],'lat':37.7857775,'lng':-122.43740055,'state':'CA','city':'San Francisco','id':'3-19282','level':3},'lt':1,'pt':0,'sort':null,'partnerId':null,'lc':[],'mode':2,'portfolio':-1,'tt':0,'ignoreLocation':false,'KeyWord':null,'rent':{'type':1,'basis':0},'term':'San Francisco, CA','PageNum':$i,'PageSize':30,'state':{'\$type':'Cityfeet.Core.Listing.MultiSearchState, Core','ProviderPosition':{'PDS':$((30 * ($i -1))),'CF':0}}}"
    content="$(curl -v -s "$url" --header "Cookie: $cookie" --header "Content-Type: application/json" --data "$body" --cookie "$cookie")"
    echo "$content" > ./data/city-feet-com-listings-spatial-$DATE-$i.json
    sleep 5
done
```
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
Or something a little more complex (with domain filter & regex [note you can split regex into multiple filters using ```||```]):
```
./gcli https://en.wikipedia.org/wiki/List_of_HTTP_status_codes en.wikipedia.org ".*List.*status_codes$"
```
Or without the domain filter:
```
./gcli https://en.wikipedia.org/wiki/List_of_HTTP_status_codes _ ".*List.*status_codes$"
```

## Running on Docker Swarm
(TODO)
