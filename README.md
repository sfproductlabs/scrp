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
### TL;DR Swarm Scripts for Hetzner
This assumes you've setup a project in Hetzner and an API key. It should be a fresh environment. We will delete ALL the machines.
On your local machine (from the scrp github repository):
```
sudo apt install hcloud-cli
hcloud ssh-key create --name andy --public-key-from-file ~/.ssh/id_rsa.pub  
hcloud network create --ip-range=10.1.0.0/16 --name=aftnet
hcloud network add-subnet --ip-range=10.1.0.0/16 --type=server --network-zone=eu-central aftnet
for n in {1..30}; do (hcloud server create --name scrp$RANDOM$RANDOM$RANDOM$RANDOM --type cx11 --image debian-9 --datacenter nbg1-dc3 --network aftnet --ssh-key andy 2>&1 >/dev/null &) ; done
watch -n 5 "echo "Press Ctrl-c to exit when your server count meets the desired amount. You will need to copy and paste just the following instructions to proceed." && hcloud server list | grep 'running' | awk 'END {print NR}'"
rm *.txt
hcloud server list -o columns=name -o noheader > scrps-names.txt
hcloud server list -o columns=ipv4 -o noheader > scrps-ips.txt
cat scrps-names.txt | xargs -I {} hcloud server describe -o json {} | jq -r '.private_net[0].ip' >> scrps-vips.txt
hcloud server create --name cassandra1 --type cx41 --image debian-9 --datacenter nbg1-dc3 --network aftnet --ssh-key andy
hcloud server describe -o json cassandra1 | jq -r '.private_net[0].ip' > cassandra-vip.txt
hcloud server create --name manager1 --type cx11 --image debian-9 --datacenter nbg1-dc3 --network aftnet --ssh-key andy
hcloud server describe -o json manager1 | jq -r '.private_net[0].ip' > manager-vip.txt
scp -o StrictHostKeyChecking=no *.txt root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
scp -o StrictHostKeyChecking=no ansible/* root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
scp -o StrictHostKeyChecking=no scrp-docker-compose.yml root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
scp -o StrictHostKeyChecking=no .setup/schema* root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
```
If it stuffs up run **DANGEROUS** it will delete all your servers for the project:
```
hcloud server list -o columns=name -o noheader | xargs -P 8 -I {} hcloud server delete {}
```
If not get on the manager node ```ssh -l root -A $(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}')``` and run:
```
apt-get update && \
apt-get upgrade -y && \
apt-get install apt-transport-https ca-certificates curl gnupg-agent software-properties-common -y && \
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo apt-key add - && \
apt-key fingerprint 0EBFCD88 && \
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable" && \
apt-get update && \
apt-get install docker-ce docker-ce-cli containerd.io ansible -y && \
docker swarm init --advertise-addr=ens10 && \
docker swarm join-token worker | xargs | sed -r 's/^.*(docker.*).*$/\1/' > join.sh && \
chmod +x join.sh && \
printf "\n[defaults]\nhost_key_checking = False\n" >> /etc/ansible/ansible.cfg && \
printf "\n[cassandras]\n" >> /etc/ansible/hosts && \
cat cassandra-vip.txt >> /etc/ansible/hosts && \
printf "\n[managers]\n" >> /etc/ansible/hosts && \
cat manager-vip.txt >> /etc/ansible/hosts && \
printf "\n[dockers]\n" >> /etc/ansible/hosts && \
cat manager-vip.txt >> /etc/ansible/hosts && \
cat scrps-vips.txt >> /etc/ansible/hosts && \
cat cassandra-vip.txt >> /etc/ansible/hosts && \
printf "\n[scrps]\n" >> /etc/ansible/hosts && \
cat scrps-vips.txt >> /etc/ansible/hosts && \
ansible dockers -a "uptime" && \
printf "\n            $(cat join.sh | awk '{print $0}')" >> swarm-init.yml && \
ansible-playbook swarm-init.yml && \
ansible dockers -a "docker stats --no-stream" && \
docker node ls && \
docker node update --label-add cassandra=true cassandra1 && \
docker network create -d overlay --attachable forenet --subnet 192.168.9.0/24 && \
docker network create -d overlay --attachable forenet --subnet 192.168.9.0/24 && \
ansible-playbook cassandras-init.yml && \
docker secret create schema.1.cql schema.1.cql && \
docker stack deploy -c scrp-docker-compose.yml scrp
```

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
$ = Your client/development machine, run them from this git repository root
# = As root on the dockermanager
d# = As root on the docker swarm drone
```

##### Setting up Hetzner
**Remember to run the $ commands from the git repository root**

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
* Create 30 servers (replace the type with your instance preference Ex. cx41)
```
$for n in {1..30}; do (hcloud server create --name scrp$RANDOM$RANDOM$RANDOM$RANDOM --type cx11 --image debian-9 --datacenter nbg1-dc3 --network aftnet --ssh-key andy &) ; done
```
* In a SEPARATE terminal see the status of your booting machines (you can delete them all using the command below if something bad happens):
```
$watch -n 5 "echo "Press Ctrl-c to exit when your server count meets the desired amount" && hcloud server list | grep 'running' | awk 'END {print NR}'"
```
* Get a list of them. IMPORTANT. This will be used to delete the servers later. Check them!
```
$rm *.txt
$hcloud server list -o columns=name -o noheader > scrps-names.txt
$hcloud server list -o columns=ipv4 -o noheader > scrps-ips.txt
$cat scrps-names.txt | xargs -I {} hcloud server describe -o json {} | jq -r '.private_net[0].ip' >> scrps-vips.txt
```
* Create a cassandra server (16GB ram):
```
$hcloud server create --name cassandra1 --type cx41 --image debian-9 --datacenter nbg1-dc3 --network aftnet --ssh-key andy
$hcloud server describe -o json cassandra1 | jq -r '.private_net[0].ip' > cassandra-vip.txt
```
* Create a manager node, copy some files to it and login:
Addtional step required *only* on a mac:
```
eval `ssh-agent`
ssh-add ~/.ssh/id_rsa
```
Now create a manager, and get to it:
```
$hcloud server create --name manager1 --type cx11 --image debian-9 --datacenter nbg1-dc3 --network aftnet --ssh-key andy
$hcloud server describe -o json manager1 | jq -r '.private_net[0].ip' > manager-vip.txt
$scp -o StrictHostKeyChecking=no *.txt root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
$scp -o StrictHostKeyChecking=no ansible/* root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
$scp -o StrictHostKeyChecking=no scrp-docker-compose.yml root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
$scp -o StrictHostKeyChecking=no .setup/schema* root@$(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}'):~/
$ssh -l root -A $(hcloud server list -o columns=ipv4,name -o noheader | grep manager1 | awk '{print $1}')
```
##### Initializing a Docker Swarm
https://docs.docker.com/engine/install/debian/

From the #docker manager1 (last ssh command above) as root **(it's important to make sure this runs perfectly)** run:

```
apt-get update && \
apt-get upgrade -y && \
apt-get install apt-transport-https ca-certificates curl gnupg-agent software-properties-common -y && \
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo apt-key add - && \
apt-key fingerprint 0EBFCD88 && \
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable" && \
apt-get update && \
apt-get install docker-ce docker-ce-cli containerd.io ansible -y && \
docker swarm init --advertise-addr=ens10 && \
docker swarm join-token worker | xargs | sed -r 's/^.*(docker.*).*$/\1/' > join.sh && \
chmod +x join.sh
```
Now we can setup docker on all the client machines using ansible (still in the docker manager1):
```
printf "\n[defaults]\nhost_key_checking = False\n" >> /etc/ansible/ansible.cfg

printf "\n[cassandras]\n" >> /etc/ansible/hosts
cat cassandra-vip.txt >> /etc/ansible/hosts

printf "\n[managers]\n" >> /etc/ansible/hosts
cat manager-vip.txt >> /etc/ansible/hosts

printf "\n[dockers]\n" >> /etc/ansible/hosts
cat manager-vip.txt >> /etc/ansible/hosts
cat scrps-vips.txt >> /etc/ansible/hosts
cat cassandra-vip.txt >> /etc/ansible/hosts

printf "\n[scrps]\n" >> /etc/ansible/hosts
cat scrps-vips.txt >> /etc/ansible/hosts
```

Test the machines are contactable:
```
ansible dockers -a "uptime"
```

If that worked, install docker on all the machines:
```
printf "\n            $(cat join.sh | awk '{print $0}')" >> swarm-init.yml
ansible-playbook swarm-init.yml
```

Test the dockers are up:
```
ansible dockers -a "docker stats --no-stream"
docker node ls
```
Now deploy the swarm stack:
```
docker node update --label-add cassandra=true cassandra1
ansible-playbook cassandras-init.yml
docker network create -d overlay --attachable forenet --subnet 192.168.9.0/24 
docker secret create schema.1.cql schema.1.cql
docker stack deploy -c scrp-docker-compose.yml scrp
```
Give it a few minutes to boot, the scrps will take a while and likely fail a few times before they finally connect to cassandra, to debug example:
```
docker service ps scrp_cassandra --no-trunc
docker service logs scrp_cassandra -f
```


##### Deleting Machines

* DELETE THEM. Yes. Let's get used to it, and make sure we know what we're doing. Double check everything before executing these commands.
```
$cat scrps-names.txt | xargs -I {} hcloud server delete {}
```
or DANGEROUS (but great for cleaning up, will include cassandra), in parallel:
```
hcloud server list -o columns=name -o noheader | xargs -P 8 -I {} hcloud server delete {}
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