package stats

import (
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
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

type Report struct {
	Host     string
	Dest     string
	RespAddr string
	FlowKey  string
	Round    uint16
	TTL      uint8
	Delay    int32
}

func (r *Report) String() string {
	return fmt.Sprintf("%10s | %20s | Resp: %20s | R: %4d | TTL: %4d | %4.2f ms",
		r.Host, r.Dest, r.RespAddr, r.Round, r.TTL, float64(r.Delay)/1000.0)
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

func MetricProcessing(rx chan *Metric, tx chan *Metric, report chan *Report, stop chan interface{}, timeout time.Duration) {
	cache := NewMap(timeout, timeout, false)
	go cache.Run()
	for {
		select {
		case e1 := <-rx:
			t, ok := cache.Load(e1.FullKey())
			if ok {
				data := t.(*Metric)
				r := &Report{
					Host:     data.Host,
					Dest:     data.Dest,
					RespAddr: e1.RespAddr,
					FlowKey:  data.Key(),
					Round:    e1.ID >> 8,
					TTL:      data.TTL,
					Delay:    int32(e1.TimeStamp.Sub(data.TimeStamp).Microseconds()),
				}

				report <- r
				cache.Delete(e1.FullKey())
			} else {
				logrus.Warn(e1.FullKey(), " not found | ", e1.ID, e1.ID>>8, "TTL:", e1.ID%256)
			}
		case e2 := <-tx:
			cache.Store(e2.FullKey(), e2, time.Now())
		case <-stop:
			break
		}
	}
}
