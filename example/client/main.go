package main

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/pio"
	"github.com/zartbot/zprobe/stats"
)

func main() {

	dst := []string{"www.cisco.com", "www.baidu.com", "www.sina.com"}

	p := pio.New("", dst, 4, 32)
	p.RoundInterval = 5 * time.Second
	p.PacketInterval = 10 * time.Millisecond
	go p.Start()

	report := make(chan *stats.Metric, 10)

	go stats.MetricProcessing(p.RecvChan, p.SendChan, report, 5*time.Second)

	for {
		e1 := <-report
		logrus.Info(e1.HostName, "|", e1.RespAddr, "|TTL:", e1.TTL, "|", e1.Delay)
	}
}
