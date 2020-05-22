#!/bin/bash
GOCQL_HOST_LOOKUP_PREFER_V4=true /usr/bin/nice -n 5 ./gsvc localhost false false ./.setup/keys/rootCa.crt ./.setup/keys/cassandra-client.crt ./.setup/keys/cassandra-client.key
#On your client
#./gcli &;