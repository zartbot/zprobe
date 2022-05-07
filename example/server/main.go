package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

func main() {
	/*
		ctx := context.Background()
		conn, err := clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			//Debug:           true,
			DialTimeout:     time.Second,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
		})

		if err != nil {
			logrus.Fatal(err)
		}
		if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
			logrus.Fatal(err)
		}
		err = conn.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS example (
				  Col1 UInt8
				, Col2 String
				, Col3 FixedString(3)
				, Col5 Map(String, UInt8)
				, Col6 Array(String)
				, Col7 Tuple(String, UInt8, Array(Map(String, String)))
				, Col8 DateTime
			) Engine = Memory
		`)
		if err != nil {
			logrus.Fatal(err)
		}

		batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
		if err != nil {
			logrus.Fatal(err)
		}
		for i := 0; i < 500_000; i++ {
			err := batch.Append(
				uint8(42),
				"ClickHouse", "Inc",
				map[string]uint8{"key": 1},             // Map(String, UInt8)
				[]string{"Q", "W", "E", "R", "T", "Y"}, // Array(String)
				[]interface{}{ // Tuple(String, UInt8, Array(Map(String, String)))
					"String Value", uint8(5), []map[string]string{
						map[string]string{"key": "value"},
						map[string]string{"key": "value"},
						map[string]string{"key": "value"},
					},
				},
				time.Now(),
			)
			if err != nil {
				logrus.Fatal(err)
			}
		}
		batch.Send()
	*/
	g := geoip.New("geoip.mmdb", "asn.mmdb")
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
		var r stats.FullReport

		err = json.Unmarshal(buf[0:n], &r)
		if err != nil {
			logrus.Warn("Parse Failed:", err)
		}
		r.Enrichment(g)
		fmt.Println(r.String())
	}
}
