package main

import (
	"github.com/gocolly/colly"
	"github.com/gocql/gocql"
	pb "github.com/sfproductlabs/scrp/src/proto"
)

//ScrapeDetail :  Actually put the data into Cassandra, will only need to change this file in the future
func ScrapeDetail(in *pb.ScrapeRequest, c *colly.Collector) {
	c.OnResponse(func(r *colly.Response) {
		db.InsertContent(in, &r.Body)
	})

}

// InsertContent will choose a random server in the cluster to write to until a successful write
func (i *Cassandra) InsertContent(in *pb.ScrapeRequest, content *[]byte) error {
	qid, err := gocql.ParseUUID(in.Id)
	if err != nil {
		qid = gocql.TimeUUID()
	}
	//Could also insert params and type
	err = i.session.Query(`INSERT INTO content (url,seq,mid,qid,raw) values (?,?,?,?,?)`,
		in.Url,
		gocql.TimeUUID(),
		GetMachineString(),
		qid,
		string(*content),
	).Exec()
	return err
}
