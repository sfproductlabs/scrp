package main

import (
	"math/rand"
	"time"

	"github.com/gocql/gocql"
	pb "github.com/sfproductlabs/scrp/src/proto"
)

//Cassandra for working with C*
type Cassandra struct {
	URLs             []string
	Username         string
	Password         string
	Keyspace         string
	UserAgent        string
	RetentionPolicy  string
	WriteConsistency string

	// Path to CA file
	SSLCA string
	// Path to host cert file
	SSLCert string
	// Path to cert key file
	SSLKey string
	// Use SSL but skip chain & host verification
	VerifyHost bool

	Retry bool

	session *gocql.Session
}

// Connect initiates the primary connection to the range of provided URLs
func (i *Cassandra) Connect() error {
	rand.Seed(time.Now().UnixNano())
	cluster := gocql.NewCluster(i.URLs...)
	cluster.Keyspace = i.Keyspace
	//cluster.ProtoVersion = 4
	cluster.Consistency = gocql.Quorum
	if i.SSLCA != "" {
		sslOpts := &gocql.SslOptions{
			CaPath:                 i.SSLCA,
			EnableHostVerification: i.VerifyHost,
		}
		if i.SSLCert != "" && i.SSLKey != "" {
			sslOpts.CertPath = i.SSLCert
			sslOpts.KeyPath = i.SSLKey
		}
		cluster.SslOpts = sslOpts
	}
	var err error
	i.session, err = cluster.CreateSession()
	if err != nil {
		return err
	}
	return nil
}

// Close will terminate the session to the backend, returning error if an issue arises
func (i *Cassandra) Close() error {
	if !i.session.Closed() {
		i.session.Close()
	}
	return nil
}

//InsertURL that the url should be downloaded
func (i *Cassandra) InsertURL(in *pb.ScrapeRequest) error {
	qid, err := gocql.ParseUUID(in.Id)
	if err != nil {
		return err
	}
	if err = i.session.Query(`INSERT INTO queries (id,domain,filter) values (?,?,?) IF NOT EXISTS`,
		qid,
		in.Domain,
		in.Filter,
	).Exec(); err == nil {
		err = i.session.Query(`INSERT INTO urls (url,status,seq,sched,completed,mid,qid,attempts) values (?,?,?,?,?,?,?,?) IF NOT EXISTS`,
			in.Url,
			0,
			gocql.TimeUUID(),
			time.Now().UTC(),
			false,
			GetMachineString(),
			qid,
			0).Exec()
	}

	return err
}

//UpdateURL either mark as done, or failed and maybe move it if too many attempts or scceeded
func (i *Cassandra) UpdateURL(in *pb.ScrapeRequest) error {
	//Success
	sched, err := time.Parse(time.Now().String(), in.Sched)
	if err != nil {
		sched = time.Now().UTC()
	}
	qid, err := gocql.ParseUUID(in.Id)
	if err != nil {
		qid = gocql.TimeUUID()
	}
	if in.Status >= 200 && in.Status < 300 {
		if err = i.session.Query(`INSERT INTO successes (url,status,seq,sched,mid,qid,size,attempts) values (?,?,?,?,?,?,?,?)`,
			in.Url,
			in.Status,
			gocql.TimeUUID(),
			sched,
			GetMachineString(),
			qid,
			in.Size,
			in.Attempts+1,
		).Exec(); err == nil {
			err = i.session.Query(`UPDATE urls set completed=? where url=?`,
				true,
				in.Url,
			).Exec()
		}
	} else {
		if in.Attempts > retries {
			if err = i.session.Query(`INSERT INTO failures (url,status,seq,sched,mid,qid,attempts) values (?,?,?,?,?,?,?)`,
				in.Url,
				in.Status,
				gocql.TimeUUID(),
				sched,
				GetMachineString(),
				qid,
				in.Attempts+1,
			).Exec(); err == nil {
				err = i.session.Query(`UPDATE urls set completed=? where url=?`,
					true,
					in.Url,
				).Exec()
			}
		} else {
			err = i.session.Query(`UPDATE urls set attempt=null, attempts=? where url=?`,
				in.Attempts+1,
				in.Url,
			).Exec()
		}
	}
	return err
}

//GetTodos Get stuff to work on
func (i *Cassandra) GetTodos() *gocql.Iter {
	return i.session.Query(`SELECT * FROM URLS where completed = false`).PageSize(200).Iter()
}

//UpdateAttempt
func (i *Cassandra) UpdateAttempt(url string) {
	i.session.Query(`UPDATE urls set attempt=? where url=?`,
		gocql.TimeUUID(),
		url,
	).Exec()
}

//GetQuery Get stuff to work on
func (i *Cassandra) GetQuery(id string) (string, string, error) {
	qid, err := gocql.ParseUUID(id)
	if err != nil {
		return "", "", err
	}
	var domain, filter string
	if err := i.session.Query(`SELECT domain, filter FROM queries WHERE id = ? LIMIT 1`,
		qid).Consistency(gocql.One).Scan(&domain, &filter); err != nil {
		return "", "", err
	}
	return domain, filter, nil
}
