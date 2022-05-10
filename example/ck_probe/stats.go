package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

var SessionDB *sync.Map

type Session struct {
	RespAddr   map[int]string
	ServerInfo []*stats.ServerInfo
	Stats      []*stats.RollingStatus
}

func ProcessRecord(ch chan *stats.Report, g *geoip.GeoIPDB) {
	for {
		r := <-ch
		go insert(r, g)
	}
}
func insert(r *stats.Report, g *geoip.GeoIPDB) {
	//find bfd session
	t, ok := SessionDB.Load(r.Key())
	if !ok {
		logrus.Warn("invalid session")
		return
	}
	s := t.(*Session)
	ttl := int(r.TTL)
	// update server info when changed
	if (r.RespAddr != "") && (s.RespAddr[ttl] != r.RespAddr) {
		s.RespAddr[ttl] = r.RespAddr
		s.ServerInfo[ttl] = stats.Enrichment(r.RespAddr, g)
	}

	if r.Loss == 1 {
		s.Stats[ttl].UpdateLoss()
	} else {
		s.Stats[ttl].Update(float64(r.Delay))
	}
	SessionDB.Store(r.Key(), s)
}

func printDB(probeName string, probList []string, maxPath int) {
	for {

		for _, dst := range probList {
			fmt.Println("===========================================================================================================================================================")
			fmt.Printf("%s---------->%s\n", probeName, dst)
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
					fmt.Printf("%2d | %20s-%-40s | (%6d)%-30s | %10s,%-20s | D:%5.2f J:%5.2f L:%5.2f\n",
						i, si.Address, si.Name, si.ASN, si.SPName,
						si.City, si.Country,
						delay, jitter, loss)
				}
				fmt.Println("--------------------------------------------------------------------------------------------------------------------------------------------------------")
			}
			fmt.Println("============================================================================================================================================================")
		}

		time.Sleep(5 * time.Second)
	}
}
