package pio

import (
	"fmt"
	"net"
	"time"

	"github.com/zartbot/zprobe/stats"
)

func (p *ProbeClient) IPv4TCPPing(dstName string, dst string, id uint16, dport uint16) {
	report := &stats.Metric{
		Host:      p.Name,
		Dest:      dstName,
		Path:      0,
		SrcAddr:   p.netSrcAddr.String(),
		DstAddr:   dst,
		SrcPort:   0,
		DstPort:   dport,
		RespAddr:  fmt.Sprintf("tcp:%s:%d", dst, dport),
		ID:        id,
		TTL:       0, //TCP Probe does not require TTL
		TimeStamp: time.Now(),
	}
	p.SendChan <- report

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", dst, dport), time.Second*2)
	if err != nil {
		return
	}

	conn.Close()
	m := &stats.Metric{
		Host:      p.Name,
		Dest:      dstName,
		Path:      0,
		SrcAddr:   p.netSrcAddr.String(),
		DstAddr:   dst,
		SrcPort:   0,
		DstPort:   dport,
		ID:        id,
		TTL:       0, //TCP Probe does not require TTL
		RespAddr:  fmt.Sprintf("tcp:%s:%d", dst, dport),
		TimeStamp: time.Now(),
	}
	p.RecvChan <- m
}
