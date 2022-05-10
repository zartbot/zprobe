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
	timeout   time.Duration
}

func New(name string, dst []string, maxPath int, maxTTL int, timeout time.Duration) *ProbeConfig {

	client := pio.New(name, "", dst, maxPath, uint8(maxTTL))

	client.RoundInterval = 1 * time.Second
	client.PacketInterval = 10 * time.Millisecond
	result := &ProbeConfig{
		client:    client,
		Report:    make(chan *stats.Report, 10),
		statsStop: make(chan interface{}),
		timeout:   timeout,
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
	go stats.MetricProcessing(p.client.SendChan, p.client.RecvChan, p.Report, p.statsStop, p.timeout)

}

func (p *ProbeConfig) Stop() {
	go p.client.Stop()
	p.statsStop <- nil
}
