package stats

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/zartbot/zprobe/geoip"
)

type FullReport struct {
	Host      string
	Dest      string
	RespAddr  string
	RespName  string
	FlowKey   string
	Round     uint16
	TTL       uint8
	Delay     uint32
	City      string
	Region    string
	Country   string
	ASN       uint32
	SPName    string
	Lat       float64
	Long      float64
	Timestamp time.Time
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
	f.ASN = uint32(geo.ASN)
	f.SPName = geo.SPName
	f.City = geo.City
	f.Region = geo.Region
	f.Country = geo.Country
	f.Lat = geo.Latitude
	f.Long = geo.Longitude

	f.Timestamp = time.Now()

}

var CKTableSchema = `
CREATE TABLE IF NOT EXISTS zprobe (
	  Host String
	, Dest String
	, RespAddr String
	, RespName String
	, FlowKey String
	, Round UInt16
	, TTL UInt8
	, Delay UInt32
	, City String
	, Region String
	, Country String
	, ASN UInt32
	, SPName String
	, Latitude Float64
	, Longitude Float64
	, Timestamp DateTime
) Engine = Memory
`
