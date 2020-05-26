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
Add backend.local to your /etc/hosts file:
```
bash -c 'echo "127.0.0.1 backend.local" >> /etc/hosts'
```

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
./gcli backend.local:50551 https://en.wikipedia.org/wiki/List_of_HTTP_status_codes
```
Or something a little more complex (with domain filter & regex [note you can split regex into multiple filters using ```||```]):
```
./gcli backend.local:50551 https://en.wikipedia.org/wiki/List_of_HTTP_status_codes en.wikipedia.org ".*List.*status_codes$"
```
Or without the domain filter and just the regex (use the _ operator to skip):
```
./gcli backend.local:50551 https://en.wikipedia.org/wiki/List_of_HTTP_status_codes _ ".*List.*status_codes$"
```

## Running on Docker Swarm
### Deploy to a swarm
*Important: First make sure you deploy the [schema](https://github.com/sfproductlabs/scrp/blob/master/.setup/schema.1.cql) to cassandra somewhere.*

Ex. ```cqlsh --ssl -f ./.setup/schema.1.cql ```


[Checkout and use the swarm-config example](https://github.com/sfproductlabs/scrp/blob/master/scrp-docker-compose.yml) then on your docker swarm manager:
```
docker stack deploy -c scrp-docker-compose.yml scrp
```
Then follow the logs to see if you need to update anything:
```
docker service logs scrp_scrp -f
```

Then issue a query to the swarm (as above):
```
docker run -it --net=forenet sfproductlabs/scrp /app/scrp/gcli scrp_scrp:50551 https://httpbin.org/delay/2
```
#### Deploying swarm on Hetzner
```
$ = Your client/development machine
# = As root on the dockermanager
d# = As root on the docker swarm drone
```

* Install hetzner cli:
```
$sudo apt install hcloud-cli
```
* Go to the cloud console and create a project (important! make sure it's a new one, we will be deleting every server in here when we are done)
* Then click on project->access->api tokens->generate token
* Setup access on your local machine to the datacenters/project:
```
$hcloud context create scrp
```
* Make sure there are no servers here (yet)
```
$hcloud server list  
```
* Add your local machine to ssh auth
```
$hcloud ssh-key create --name andy --public-key-from-file ~/.ssh/id_rsa.pub  
```
* Choose a server-type
```
$hcloud server-type list
$hcloud image list
$hcloud datacenter list
```
* Create a network
```
$hcloud network create --ip-range=10.1.0.0/16 --name=aftnet
$hcloud network add-subnet --ip-range=10.1.0.0/16 --type=server --network-zone=eu-central aftnet
```
* Create 100 servers
```
$for n in {1..2}; do hcloud server create --name scrp$n --type cx11 --image ubuntu-20.04 --datacenter nbg1-dc3 --network aftnet --ssh-key andy; done
```
* Get a list of them. IMPORTANT. This will be used to delete the servers later. Check them!
```
$rm scrps-vips.txt
$hcloud server list -o columns=name -o noheader > scrps-names.txt
$cat scrps-names.txt | xargs -I {} hcloud server describe -o json {} | jq -r '.private_net[0].ip' >> scrps-vips.txt
```
* Create a cassandra server
```
$hcloud server create --name cassandra1 --type cx41 --image ubuntu-20.04 --datacenter nbg1-dc3 --network aftnet --ssh-key andy
```

* DELETE THEM. Yes. Let's get used to it, and make sure we know what we're doing.
```
$cat scrps-names.txt | xargs -I {} hcloud server delete {}
```
or DANGEROUS (but great for cleaning up, will include cassandra):
```
$hcloud server list -o columns=name -o noheader | xargs -I {} hcloud server delete {}
```





##### Misc
Example commands (https://docs.hetzner.cloud/):
```
$source <(hcloud completion bash)   # bash
$source <(hcloud completion zsh)   # zsh
$hcloud server list
$hcloud ssh-key create --name demo --public-key-from-file ~/.ssh/id_rsa.pub         
$hcloud server create --name demoserver --type cx11 --image debian-9 --ssh-key demo                 
$hcloud server list             
$hcloud server list | grep -E "[0-9]+.[0-9]+.[0-9]+.[0-9]+" | sed -r 's/.*(\w[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+).*$/\1/' > scrps-ips.txt
$hcloud server list | grep -E '^[^ID]' | sed -r 's/^[0-9]+ +([^ ]+).*$/\1/ig' > scrps-names.txt
```