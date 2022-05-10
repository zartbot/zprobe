package stats

import (
	"fmt"
	"net"
	"time"
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
