package stats

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/geoip"
)

type ProbeInfo struct {
	SrcAddr net.IP
	DstAddr net.IP
	SrcPort uint16
	DstPort uint16
	TTL     uint8
	ID      uint16
}

type Metric struct {
	Path      int
	Host      string
	Dest      string
	SrcAddr   string
	DstAddr   string
	SrcPort   uint16
	DstPort   uint16
	TTL       uint8
	ID        uint16
	RespAddr  string
	TimeStamp time.Time
	Delay     time.Duration
}

func (m *Metric) FullKey() string {
	return fmt.Sprintf("%s:%d,%s:%d-%d",
		m.SrcAddr, m.SrcPort,
		m.DstAddr, m.DstPort,
		m.ID)
}

func (m *Metric) Key() string {
	return fmt.Sprintf("%s:%d,%s:%d",
		m.SrcAddr, m.SrcPort,
		m.DstAddr, m.DstPort)
}

type Report struct {
	Path     int
	Host     string
	Dest     string
	DestAddr string
	RespAddr string
	FlowKey  string
	Round    uint16
	TTL      uint8
	Delay    int32
	Loss     uint8
}

func (r *Report) String() string {
	return fmt.Sprintf("%10s | %20s | Resp: %20s | R: %4d | TTL: %4d | %4.2f ms | Loss: %d",
		r.Host, r.Dest, r.RespAddr, r.Round, r.TTL, float64(r.Delay)/1000.0, r.Loss)
}

func (r *Report) Key() string {
	return fmt.Sprintf("%s:%s:%d",
		r.Host, r.Dest, r.Path)
}

func MetricProcessing(tx chan *Metric, rx chan *Metric, report chan *Report, stop chan interface{}, timeout time.Duration) {
	cache := NewSessionTable(timeout, timeout, false)
	for {
		select {
		case t0 := <-cache.TimeOutChan:
			r := &Report{
				Host:     t0.Host,
				Dest:     t0.Dest,
				Path:     t0.Path,
				DestAddr: t0.DstAddr,
				RespAddr: "",
				FlowKey:  t0.Key(),
				Round:    t0.ID >> 8,
				TTL:      t0.TTL,
				Delay:    0,
				Loss:     1,
			}
			report <- r
		case e1 := <-rx:
			t, ok := cache.Load(e1.FullKey())
			if ok {
				data := t.(*Metric)
				r := &Report{
					Host:     data.Host,
					Dest:     data.Dest,
					DestAddr: data.DstAddr,
					Path:     data.Path,
					RespAddr: e1.RespAddr,
					FlowKey:  data.Key(),
					Round:    e1.ID >> 8,
					TTL:      data.TTL,
					Delay:    int32(e1.TimeStamp.Sub(data.TimeStamp).Microseconds()),
					Loss:     0,
				}
				report <- r
				cache.Delete(e1.FullKey())
			}
			/*
				else {
					logrus.Warn(e1.FullKey(), " not found | ", e1.ID, e1.ID>>8, "TTL:", e1.ID%256)
				}*/
		case e2 := <-tx:
			cache.Store(e2.FullKey(), e2, time.Now())
		case <-stop:
			break
		}
	}
}

type Session struct {
	DestAddr   string
	RespAddr   map[int]string
	ServerInfo []*ServerInfo
	Stats      []*RollingStatus
	InitFlag   []uint8
	Lock       *sync.RWMutex
}

func ProcessRecord(ch chan *Report, g *geoip.GeoIPDB, SessionDB *sync.Map, DelayWinSize int) {
	for {
		r := <-ch
		go insert(r, g, SessionDB, DelayWinSize)
	}
}
func insert(r *Report, g *geoip.GeoIPDB, SessionDB *sync.Map, DelayWinSize int) {
	//find bfd session
	t, ok := SessionDB.Load(r.Key())
	if !ok {
		logrus.Warn("invalid session")
		return
	}
	s := t.(*Session)
	ttl := int(r.TTL)

	s.Lock.Lock()
	if s.DestAddr == "" {
		s.DestAddr = r.DestAddr
	}
	// update server info when changed
	if (r.RespAddr != "") && (s.RespAddr[ttl] != r.RespAddr) {
		s.RespAddr[ttl] = r.RespAddr
		s.ServerInfo[ttl] = Enrichment(r.RespAddr, g)
	}
	if r.Loss == 1 {
		s.Stats[ttl].UpdateWeightedLoss()
	} else {
		s.Stats[ttl].Update(float64(r.Delay))
		if s.InitFlag[ttl] == 0 {
			for i := 0; i < DelayWinSize; i++ {
				s.Stats[ttl].UpdateWeighted(float64(r.Delay))
			}
			s.InitFlag[ttl] = 1
		}
	}
	s.Lock.Unlock()
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

func NewSessionDB(probeName string, probeList []string, maxPath int, maxTTL int, DelayWinSize int, LossWinSize int) *sync.Map {
	SessionDB := &sync.Map{}
	for _, dst := range probeList {
		for i := 0; i <= maxPath; i++ {
			key := fmt.Sprintf("%s:%s:%d", probeName, dst, i)
			db := &Session{
				RespAddr:   make(map[int]string),
				ServerInfo: make([]*ServerInfo, maxTTL+1),
				Stats:      make([]*RollingStatus, maxTTL+1),
				InitFlag:   make([]uint8, maxTTL+1),
				Lock:       new(sync.RWMutex),
			}
			for j := 0; j <= maxTTL; j++ {
				db.RespAddr[j] = ""
				db.ServerInfo[j] = &ServerInfo{}
				db.Stats[j] = NewRollingStatus(DelayWinSize, LossWinSize)
				db.InitFlag[j] = 0
				//delayWinSize need > 31 for jitter accuracy
				//loss is 64bits bitmap with 1/64 accuracy
			}
			SessionDB.Store(key, db)
		}
	}
	return SessionDB
}

func PrintDB(SessionDB *sync.Map, probeName string, probList []string, maxPath int) {
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
				data.Lock.RLock()

				for i := 0; i < len(data.Stats); i++ {
					if data.RespAddr[i] == "" {
						continue
					}

					si := data.ServerInfo[i]
					sd := data.Stats[i]
					delay, jitter, loss := sd.GetWeighted(2)

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
				data.Lock.RUnlock()
			}
			table.Render()
		}

		time.Sleep(5 * time.Second)
	}
}
