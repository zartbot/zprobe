package main

import (
	"fmt"
	"time"

	"github.com/zartbot/zprobe"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

func main() {

	dst := []string{"www.sina.com"}

	p := zprobe.New("zartbot", dst, 4, 32, 2*time.Second)
	p.SetPacketInterval(1 * time.Millisecond)
	p.SetRoundInterval(10 * time.Second)
	go p.Start()

	g := geoip.New("../geoip/geoip.mmdb", "../geoip/asn.mmdb")

	for {

		r := <-p.Report

		i := stats.Enrichment(r.RespAddr, g)
		fmt.Printf("%10s | %20s %4d | Resp: %30s %-40s| %20s | %6d:%-20s R: %4d | TTL: %4d | %4.2f ms | Loss: %d\n",
			r.Host, r.Dest, r.Path, r.RespAddr, i.Name, i.Country, i.ASN, i.SPName, r.Round, r.TTL, float64(r.Delay)/1000.0, r.Loss)
	}

}
