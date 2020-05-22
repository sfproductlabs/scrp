-- To run:
-- cqlsh --ssl -f schema.1.cql 

-- SCRP
-- drop keyspace scrp;
create keyspace scrp WITH REPLICATION = {  'class':'NetworkTopologyStrategy', 'DC1':'1' };

use scrp;

create table queries (
  id timeuuid,
  domain text,
  filter text,
  PRIMARY KEY (id)
);

create table urls (
  url text,
  seq timeuuid, --sequence it came in/last updated
  status int, --0 untouched, 200=OK etc
  sched timestamp, --scheduled for processing
  mid text, --machine id
  qid timeuuid, --query id
  attempts int,
  PRIMARY KEY (url)
);
CREATE INDEX urls_qid_idx ON urls ( qid );

create table failures (
  url text,
  seq timeuuid, --sequence it came in/last updated
  status int, --0 untouched, 200=OK etc
  sched timestamp, --scheduled for processing
  mid text, --machine id
  qid timeuuid, --query ids
  attempts int,
  PRIMARY KEY (url)
);


create table successes (
  url text,
  seq timeuuid, --sequence it came in/last updated
  status int, --0 untouched, 200=OK etc
  sched timestamp, --scheduled for processing
  mid text, --machine id
  qid timeuuid, --query id
  size bigint,
  attempts int,
  PRIMARY KEY (url)
);

--Successful scrapes
create table content (
    url text,
    seq timeuuid, --sequence it came in/last updated
    mid text, --machine id
    qid timeuuid, --query id
    type text, --type of data
    params frozen<map<text,text>>,
    raw text,
    PRIMARY KEY (url)
);
CREATE INDEX content_type_idx ON content ( type );