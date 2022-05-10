package stats

import (
	"bytes"
	"net"
	"strings"
	"time"

	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/tsyncmap"
)

var EnrichmentCacheDB *tsyncmap.Map

func init() {
	EnrichmentCacheDB = &tsyncmap.Map{
		Timeout:   86400,
		CheckFreq: 720,
		Verbose:   false,
	}
	go EnrichmentCacheDB.Run()
}

type ServerInfo struct {
	Address string
	Name    string
	City    string
	Region  string
	Country string
	ASN     uint32
	SPName  string
	Lat     float64
	Long    float64
}

func Enrichment(dst string, g *geoip.GeoIPDB) *ServerInfo {

	if dst == "" {
		return &ServerInfo{}
	}
	record, valid := EnrichmentCacheDB.Load(dst)
	if valid {
		return record.(*ServerInfo)
	} else {
		addr := dst
		if strings.HasPrefix(dst, "tcp") {
			addr = strings.Split(dst, ":")[1]
		}
		rA, _ := net.LookupAddr(addr)
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
		geo := g.Lookup(addr)

		result := &ServerInfo{
			Address: addr,
			Name:    buf.String(),
			City:    geo.City,
			Region:  geo.Region,
			Country: geo.Country,
			ASN:     uint32(geo.ASN),
			SPName:  geo.SPName,
			Lat:     geo.Latitude,
			Long:    geo.Longitude,
		}
		EnrichmentCacheDB.Store(dst, result, time.Now())
		return result
	}
}
