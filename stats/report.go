package stats

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"github.com/zartbot/zprobe/geoip"
)

type FullReport struct {
	Host     string
	Dest     string
	RespAddr string
	RespName string
	FlowKey  string
	Round    uint16
	TTL      uint8
	Delay    int32
	City     string
	Country  string
	ASN      uint
	SPName   string
	Lat      float64
	Long     float64
}

func (r *FullReport) String() string {
	return fmt.Sprintf("%10s | %20s | Resp: %20s  %40s | %10s SP:(%8d)%60s | R: %4d | TTL: %4d | %4.2f ms",
		r.Host, r.Dest, r.RespAddr, r.RespName, r.City, r.ASN, r.SPName, r.Round, r.TTL, float64(r.Delay)/1000.0)
}

func (f *FullReport) Enrichment(g *geoip.GeoIPDB) {
	rA, _ := net.LookupAddr(f.RespAddr)
	var buf bytes.Buffer
	for _, item := range rA {
		if len(item) > 0 {
			//some platform may add dot in suffix
			item = strings.TrimSuffix(item, ".")
			if !strings.HasSuffix(item, ".in-addr.arpa") {
				buf.WriteString(item)
			}
		}
	}
	f.RespName = buf.String()

	geo := g.Lookup(f.RespAddr)
	f.ASN = geo.ASN
	f.SPName = geo.SPName
	f.City = geo.City
	f.Country = geo.Country
	f.Lat = geo.Latitude
	f.Long = geo.Longitude

}
