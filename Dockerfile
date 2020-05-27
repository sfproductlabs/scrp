
####################################################################################
# Scrp
####################################################################################

FROM debian:latest
EXPOSE 50551

# update packages and install required ones
RUN apt update && apt upgrade -y && apt install -y --no-install-recommends \
#  golang \
#  git \
#  libssl-dev \
#  python-pip \
  curl \
  dnsutils \
  jq \
  ca-certificates \
  valgrind \
  && apt autoclean -y \
  && apt autoremove -y \
  && rm -rf /var/lib/apt/lists/* 


####################################################################################

# ulimit increase (set in docker templats/aws ecs-task-definition too!!)
RUN bash -c 'echo "root hard nofile 1048575" >> /etc/security/limits.conf' \
 && bash -c 'echo "root soft nofile 1048575" >> /etc/security/limits.conf' \
 && bash -c 'echo "* hard nofile 1048575" >> /etc/security/limits.conf' \
 && bash -c 'echo "* soft nofile 1048575" >> /etc/security/limits.conf'

# ip/tcp tweaks, disable ipv6
RUN bash -c 'echo "net.core.somaxconn = 1048575" >> /etc/sysctl.conf' \
 && bash -c 'echo "net.ipv4.tcp_max_tw_buckets = 1440000" >> /etc/sysctl.conf' \
 && bash -c 'echo "net.ipv4.tcp_window_scaling = 1" >> /etc/sysctl.conf' \
 && bash -c 'echo "net.ipv4.tcp_syncookies = 1" >> /etc/sysctl.conf' \
 && bash -c 'echo "net.ipv4.tcp_max_syn_backlog = 1048575" >> /etc/sysctl.conf' \
 && bash -c 'echo "fs.file-max=1048575" >> /etc/sysctl.conf' \
 && bash -c 'echo "vm.max_map_count=1048575" >> /etc/sysctl.conf'

####################################################################################


WORKDIR /app/scrp
ADD . /app/scrp

ENV CASSANDRAS=cassandra1,cassandra2,cassandra3
ENV CASSANDRA_RETRY=false
ENV CASSANDRA_VERIFY_HOSTS=false
ENV CASSANDRA_ROOTCA=
ENV CASSANDRA_CLIENT_CERT=
ENV CASSANDRA_CLIENT_KEY=
ENV BACKEND_CERT=/app/scrp/.setup/keys/scrp_scrp.cert
ENV BACKEND_KEY=/app/scrp/.setup/keys/scrp_scrp.key
ENV CLUSTER_ENDPOINT=scrp_scrp:50551

ENV GOCQL_HOST_LOOKUP_PREFER_V4=true

RUN bash -c 'echo "127.0.0.1 backend.local" >> /etc/hosts'
RUN bash -c 'echo "127.0.0.1 frontend.local" >> /etc/hosts'

####################################################################################


CMD /usr/bin/nice -n 5 /app/scrp/gsvc $CASSANDRAS $CASSANDRA_RETRY $CASSANDRA_VERIFY_HOSTS $CASSANDRA_ROOTCA $CASSANDRA_CLIENT_CERT $CASSANDRA_CLIENT_KEY
