package main

import (
	"encoding/json"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/stats"
)

func RecvReport(port int, ch chan *stats.FullReport) {

	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0"),
		Port: port,
	}
	sock, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		logrus.Fatal("Listen failed: ", err)
	}

	buf := make([]byte, 1500)

	for {
		n, _, _ := sock.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		var r stats.FullReport

		err = json.Unmarshal(buf[0:n], &r)
		if err != nil {
			continue
		}
		ch <- &r
	}
}
