package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/stats"
)

func main() {
	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 1234,
	}
	sock, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		logrus.Fatal("Listen failed: ", err)
	}

	buf := make([]byte, 1500)

	for {
		n, _, err := sock.ReadFromUDP(buf)
		if err != nil {
			logrus.Warn("ReadErr:", err)
		}
		var r stats.Report

		err = json.Unmarshal(buf[0:n], &r)
		if err != nil {
			logrus.Warn("Parse Failed:", err)
		}
		fmt.Println(r.String())

	}
}
