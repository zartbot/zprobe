package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

var SessionDB *sync.Map

type Session struct {
	DestAddr   string
	RespAddr   map[int]string
	ServerInfo []*stats.ServerInfo
	Stats      []*stats.RollingStatus
	InitFlag   []uint8
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
	if s.DestAddr == "" {
		s.DestAddr = r.DestAddr
	}
	// update server info when changed
	if (r.RespAddr != "") && (s.RespAddr[ttl] != r.RespAddr) {
		s.RespAddr[ttl] = r.RespAddr
		s.ServerInfo[ttl] = stats.Enrichment(r.RespAddr, g)
	}
	if r.Loss == 1 {
		s.Stats[ttl].UpdateLoss()
	} else {
		s.Stats[ttl].Update(float64(r.Delay))
		if s.InitFlag[ttl] == 0 {
			for i := 0; i < DelayWinSize; i++ {
				s.Stats[ttl].Update(float64(r.Delay))
			}
			s.InitFlag[ttl] = 1
		}
	}
	SessionDB.Store(r.Key(), s)
}

func GetColorByLatency(latency float64) tablewriter.Colors {
	if latency < 20 {
		return tablewriter.Colors{tablewriter.FgHiGreenColor}
	}
	if latency > 150 {
		return tablewriter.Colors{tablewriter.FgHiRedColor}
	}
	if latency > 100 {
		return tablewriter.Colors{tablewriter.FgHiYellowColor}
	}
	return tablewriter.Colors{}
}

func GetColorByLoss(loss float64) tablewriter.Colors {
	if loss < 0.5 {
		return tablewriter.Colors{tablewriter.FgHiGreenColor}
	}
	if loss > 10 {
		return tablewriter.Colors{tablewriter.FgHiRedColor}
	}
	if loss > 3 {
		return tablewriter.Colors{tablewriter.FgHiYellowColor}
	}
	return tablewriter.Colors{}
}

func printDB(probeName string, probList []string, maxPath int) {
	ColorNormal := tablewriter.Colors{}

	for {
		for _, dst := range probList {
			fmt.Printf("%s---->%s\n", probeName, dst)
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"TTL  ", "Server", "Name", "City", "Country", "ASN", "SP", "Latency", "Jitter", "Loss"})
			table.SetAutoFormatHeaders(false)

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

					pTTL := fmt.Sprintf("%4d", i)
					pServer := fmt.Sprintf("%-20s", si.Address)
					pName := fmt.Sprintf("%-30.30s", si.Name)
					pCity := fmt.Sprintf("%-16.16s", si.City)
					pCountry := fmt.Sprintf("%-16.16s", si.Country)
					pASN := fmt.Sprintf("%-10d", si.ASN)
					pSP := fmt.Sprintf("%-30.30s", si.SPName)
					pLatency := fmt.Sprintf("%8.2fms", delay/1000)
					pJitter := fmt.Sprintf("%8.2fms", jitter/1000)

					pData := []string{pTTL, pServer, pName, pCity, pCountry, pASN, pSP, pLatency, pJitter, fmt.Sprintf("%4.1f%%", loss*100)}
					rowColor := make([]tablewriter.Colors, len(pData))
					for i := 0; i < len(pData); i++ {
						rowColor[i] = ColorNormal
					}
					rowColor[7] = GetColorByLatency(delay / 1000)
					rowColor[8] = GetColorByLatency(jitter / 1000)
					rowColor[9] = GetColorByLoss(loss * 100)
					table.Rich(pData, rowColor)

					if data.RespAddr[i] == data.DestAddr {
						break
					}
				}
			}
			table.Render()
		}

		time.Sleep(5 * time.Second)
	}
}
