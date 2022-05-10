package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/zartbot/zprobe"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

func main() {
	SessionDB = &sync.Map{}

	probeName := "zartbot"
	probList := []string{"www.sina.com", "202.120.58.161", "www.cisco.com"}
	maxPath := 4
	maxTTL := 32

	//create SessionDB
	for _, dst := range probList {
		for i := 0; i <= maxPath; i++ {
			key := fmt.Sprintf("%s:%s:%d", probeName, dst, i)
			db := &Session{
				RespAddr:   make(map[int]string),
				ServerInfo: make([]*stats.ServerInfo, maxTTL+1),
				Stats:      make([]*stats.RollingStatus, maxTTL+1),
			}
			for j := 0; j <= maxTTL; j++ {
				db.RespAddr[j] = ""
				db.ServerInfo[j] = &stats.ServerInfo{}
				db.Stats[j] = stats.NewRollingStatus(32, 64)
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

	go printDB(probeName, probList, maxPath)
	for {
		r := <-p.Report
		go ProcessingRecord(r, g)

	}

}
