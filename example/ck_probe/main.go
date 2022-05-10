package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

var CKTableSchema = `
CREATE TABLE IF NOT EXISTS zprobe (
	  Host String
	, Dest String
	, RespAddr String
	, RespName String
	, FlowKey String
	, TTL UInt8
	, City String
	, Region String
	, Country String
	, ASN UInt32
	, SPName String
	, Latitude Float64
	, Longitude Float64
	, Delay Float64
	, Jitter Float64
	, Loss Float64
	, Timestamp DateTime
) Engine = MergeTree
PARTITION BY (toStartOfHour(Timestamp))
ORDER BY (Host,Dest,TTL,FlowKey)
`
var DelayWinSize int = 32
var LossWinSize int = 64

func main() {
	SessionDB = &sync.Map{}

	probeName := "zartbot"
	probList := []string{"www.sina.com", "www.baidu.com", "www.tencent.com", "www.taobao.com", "www.cisco.com", "www.github.com", "www.google.com", "www.facebook.com", "www.twitter.com", "www.amazon.com"}
	//probList = []string{"www.amazon.com"}
	maxPath := 4
	maxTTL := 32
	ckAddress := "127.0.0.1:9000"
	ckUsername := "default"
	ckPassword := ""

	//create SessionDB
	for _, dst := range probList {
		for i := 0; i <= maxPath; i++ {
			key := fmt.Sprintf("%s:%s:%d", probeName, dst, i)
			db := &Session{
				RespAddr:   make(map[int]string),
				ServerInfo: make([]*stats.ServerInfo, maxTTL+1),
				Stats:      make([]*stats.RollingStatus, maxTTL+1),
				InitFlag:   make([]uint8, maxTTL+1),
			}
			for j := 0; j <= maxTTL; j++ {
				db.RespAddr[j] = ""
				db.ServerInfo[j] = &stats.ServerInfo{}
				db.Stats[j] = stats.NewRollingStatus(DelayWinSize, LossWinSize)
				db.InitFlag[j] = 0
				//delayWinSize need > 31 for jitter accuracy
				//loss is 64bits bitmap with 1/64 accuracy
			}
			SessionDB.Store(key, db)
		}
	}

	p := zprobe.New(probeName, probList, maxPath, maxTTL, 2*time.Second)
	p.SetPacketInterval(5 * time.Millisecond)
	p.SetRoundInterval(5 * time.Second)

	go p.Start()
	g := geoip.New("../geoip/geoip.mmdb", "../geoip/asn.mmdb")

	go ProcessRecord(p.Report, g)
	go printDB(probeName, probList, maxPath)

	ticker := time.NewTicker(60 * time.Second)

	//Create ClickHouse
	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{ckAddress},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: ckUsername,
			Password: ckPassword,
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
	err = conn.Exec(ctx, CKTableSchema)
	if err != nil {
		logrus.Fatal(err)
	}

	for {
		<-ticker.C
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO zprobe")
		if err != nil {
			logrus.Info("prepare batch fail:", err)
			continue
		}

		for _, dst := range probList {
			for i := 0; i <= maxPath; i++ {
				key := fmt.Sprintf("%s:%s:%d", probeName, dst, i)
				tmp, ok := SessionDB.Load(key)
				if !ok {
					continue
				}
				data := tmp.(*Session)
				for i := 0; i < len(data.Stats); i++ {
					if data.RespAddr[i] == "" {
						continue
					}
					si := data.ServerInfo[i]
					sd := data.Stats[i]
					delay, jitter, loss := sd.Get()
					err := batch.Append(
						probeName,
						dst,
						si.Address, si.Name,
						key,
						uint8(i),
						si.City, si.Region, si.Country,
						si.ASN, si.SPName,
						si.Lat, si.Long,
						delay,
						jitter,
						loss,
						time.Now(),
					)
					if err != nil {
						logrus.Warn("batch insertion fail:", err)
					}
					if data.RespAddr[i] == data.DestAddr {
						break
					}
				}
			}
		}
		batch.Send()
	}

}
