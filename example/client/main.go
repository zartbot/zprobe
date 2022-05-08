package main

import (
	"encoding/json"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe"
)

func main() {

	dst := []string{"www.sina.com", "www.taobao.com", "www.cisco.com", "www.github.com"}
	service := "127.0.0.1:1234"

	p := zprobe.New("zartbot", dst, 8, 32)
	p.SetPacketInterval(50 * time.Millisecond)
	p.SetRoundInterval(10 * time.Second)
	go p.Start()

	RemoteAddr, err := net.ResolveUDPAddr("udp", service)
	conn, err := net.DialUDP("udp", nil, RemoteAddr)
	if err != nil {
		logrus.Fatal("create connection failed: ", err)
	}
	defer conn.Close()

	for {
		e1 := <-p.Report
		//fmt.Printf("%s\n", e1.String())
		j, _ := json.Marshal(e1)
		conn.Write(j)
	}
}
