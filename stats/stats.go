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
	HostName  string
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

func MetricProcessing(rx chan *Metric, tx chan *Metric, report chan *Metric, timeout time.Duration) {
	cache := NewMap(timeout, timeout, false)

	go cache.Run()
	for {
		select {
		case e1 := <-rx:
			t, ok := cache.Load(e1.FullKey())
			if ok {
				data := t.(*Metric)
				data.Delay = e1.TimeStamp.Sub(data.TimeStamp)
				data.RespAddr = e1.RespAddr
				report <- data
				cache.Delete(e1.FullKey())
			} else {
				logrus.Warn(e1.FullKey(), "not found")
			}
		case e2 := <-tx:
			cache.Store(e2.FullKey(), e2, time.Now())
		}
	}

}
