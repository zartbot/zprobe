package zprobe

import (
	"time"

	"github.com/zartbot/zprobe/pio"
	"github.com/zartbot/zprobe/stats"
)

type ProbeConfig struct {
	client *pio.ProbeClient
	Report chan *stats.Report

	statsStop chan interface{}
}

func New(name string, dst []string, maxPath int, maxTTL int) *ProbeConfig {

	client := pio.New(name, "", dst, maxPath, uint8(maxTTL))

	client.RoundInterval = 1 * time.Second
	client.PacketInterval = 10 * time.Millisecond
	result := &ProbeConfig{
		client:    client,
		Report:    make(chan *stats.Report, 10),
		statsStop: make(chan interface{}),
	}
	return result
}

func (p *ProbeConfig) SetPacketInterval(t time.Duration) {
	p.client.PacketInterval = t
}

func (p *ProbeConfig) SetRoundInterval(t time.Duration) {
	p.client.RoundInterval = t
}

func (p *ProbeConfig) Start() {
	go p.client.Start()

	go stats.MetricProcessing(p.client.RecvChan, p.client.SendChan, p.Report, p.statsStop, 2*time.Second)

}

func (p *ProbeConfig) Stop() {
	go p.client.Stop()
	p.statsStop <- nil
}
