package main

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

func main() {

	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		//Debug:           true,
		DialTimeout:     time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	/* drop existing table for test only
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS zprobe`); err != nil {
		logrus.Fatal(err)
	}
	*/

	err = conn.Exec(ctx, stats.CKTableSchema)
	if err != nil {
		logrus.Fatal(err)
	}

	ch := make(chan *stats.FullReport, 100)
	go RecvReport(1234, ch)
	g := geoip.New("geoip.mmdb", "asn.mmdb")
	ticker := time.NewTicker(1 * time.Second)

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO zprobe")
	for {
		select {
		case r := <-ch:
			r.Enrichment(g)
			//fmt.Println(r.String())
			err := batch.Append(
				r.Host,
				r.Dest,
				r.RespAddr, r.RespName,
				r.FlowKey,
				r.Round,
				r.TTL,
				r.Delay,
				r.City, r.Region, r.Country,
				r.ASN, r.SPName,
				r.Lat, r.Long, r.Timestamp,
			)
			if err != nil {
				logrus.Warn("batch insertion fail:", err)
			}
		case <-ticker.C:
			batch.Send()
			batch, err = conn.PrepareBatch(ctx, "INSERT INTO zprobe")
			if err != nil {
				logrus.Warn(err)
			}
		}
	}

}
