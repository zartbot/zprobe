package main

import (
	"context"
	"fmt"
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

func main() {

	probeName := "zartbot"
	probeList := []string{
		"www.baidu.com",
		"www.tencent.com",
		"cn.taobao.com",
		"www.jd.com",
		"www.weibo.com",
		"www.163.com.cn",
		"www.sohu.com",
		"www.sina.com.cn",
		"www.qq.com",
		"www.youku.com",
		"www.zhihu.com",
		"www.iqiyi.com",
		"www.bilibili.com",
		"www.douban.com",
		"www.sjtu.edu.cn",
		"www.tsinghua.edu.cn",
		"www.mit.edu",
		"www.online.sh.cn",
		"www.cisco.com",
		"www.github.com",
		"www.google.com",
		"www.facebook.com",
		"www.twitter.com",
		"www.amazon.com",
		"www.aws.com",
		"www.netflix.com",
		"www.ebay.com",
		"www.office365.com",
		"www.salesforce.com",
		"video.huawan.com",
		"www.zoom.com",
		"www.webex.com",
		"meeting.tencent.com",
	}

	maxPath := 8
	maxTTL := 32

	var DelayWinSize int = 32
	var LossWinSize int = 64

	ckAddress := "127.0.0.1:9000"
	ckUsername := "default"
	ckPassword := ""

	//create SessionDB
	SessionDB := stats.NewSessionDB(probeName, probeList, maxPath, maxTTL, DelayWinSize, LossWinSize)

	p := zprobe.New(probeName, probeList, maxPath, maxTTL, 2*time.Second)
	p.SetPacketInterval(5 * time.Millisecond)
	p.SetRoundInterval(5 * time.Second)

	go p.Start()
	g := geoip.New("../geoip/geoip.mmdb", "../geoip/asn.mmdb")

	go stats.ProcessRecord(p.Report, g, SessionDB, DelayWinSize)
	go stats.PrintDB(SessionDB, probeName, probeList, maxPath)

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

		for _, dst := range probeList {
			for i := 0; i <= maxPath; i++ {
				key := fmt.Sprintf("%s:%s:%d", probeName, dst, i)
				tmp, ok := SessionDB.Load(key)
				if !ok {
					continue
				}
				data := tmp.(*stats.Session)
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
