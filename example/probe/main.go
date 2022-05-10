package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/zartbot/zprobe"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

var DelayWinSize int = 32
var LossWinSize int = 64

func main() {
	SessionDB = &sync.Map{}

	probeName := "zartbot"
	probList := []string{"www.sina.com", "www.baidu.com", "www.tencent.com", "www.taobao.com", "www.cisco.com", "www.github.com", "www.google.com", "www.facebook.com", "www.twitter.com", "www.amazon.com"}
	//probList = []string{"www.amazon.com"}
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
	printDB(probeName, probList, maxPath)

}
