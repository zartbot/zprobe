package pio

import (
	"encoding/binary"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/stats"
	"golang.org/x/net/ipv4"
)

type ProbeClient struct {
	SrcAddr        string
	Dest           []string
	MaxPath        int
	MaxTTL         uint8
	Protocol       string
	PacketInterval time.Duration
	RoundInterval  time.Duration
	SendChan       chan *stats.Metric
	RecvChan       chan *stats.Metric
	SrcPort        []uint16
	DstPort        []uint16

	recvICMPConn *net.IPConn
	netSrcAddr   net.IP
	netDstAddr   []net.IP
	af           string

	stopSignal *int32 //atomic Counters,stop when cnt =1

}

func (p *ProbeClient) validateSrcAddress() error {
	if p.SrcAddr != "" {
		addr, err := net.ResolveIPAddr(p.af, p.SrcAddr)
		if err != nil {
			return err
		}
		p.netSrcAddr = addr.IP
		return nil
	}

	//if config does not specify address, fetch local address
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		logrus.Fatal("check source addr:", err)
	}
	result := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()
	p.netSrcAddr = result.IP
	return nil
}

func (p *ProbeClient) VerifyCfg() {
	for idx, dip := range p.Dest {
		rAddr, err := net.LookupIP(dip)
		if err != nil {
			logrus.Fatal("dst address validation:", err)
		}
		p.netDstAddr[idx] = rAddr[0]
	}

	//update address family
	p.af = "ip4"

	//verify source address
	err := p.validateSrcAddress()
	if err != nil {
		logrus.Fatal("validate source address failed:", err)
	}

	var sig int32 = 0
	p.stopSignal = &sig
	atomic.StoreInt32(p.stopSignal, 0)

	if p.MaxPath > 16 {
		logrus.Fatal("Only support max ECMP = 16")
	}
	if p.MaxTTL > 64 {
		logrus.Warn("Large TTL may cause low performance")
	}

}

func New(src string, destList []string, maxPath int, maxTTL uint8) *ProbeClient {

	result := &ProbeClient{
		SrcAddr:        src,
		Dest:           destList,
		Protocol:       "udp",
		MaxPath:        maxPath,
		MaxTTL:         maxTTL,
		PacketInterval: time.Duration(100 * time.Microsecond),
		RoundInterval:  time.Duration(1 * time.Second),
		SendChan:       make(chan *stats.Metric, 10),
		RecvChan:       make(chan *stats.Metric, 10),

		netDstAddr: make([]net.IP, len(destList)),
	}

	//build port pair
	result.SrcPort = make([]uint16, maxPath)
	result.DstPort = make([]uint16, maxPath)
	for i := 0; i < maxPath; i++ {
		result.SrcPort[i] = uint16(1000 + rand.Int31n(500))
		result.DstPort[i] = uint16(33434 + rand.Int31n(64))
	}

	result.VerifyCfg()
	return result
}

func (p *ProbeClient) Recv() {
	laddr := &net.IPAddr{IP: p.netSrcAddr}
	var err error
	p.recvICMPConn, err = net.ListenIP("ip4:icmp", laddr)
	if err != nil {
		logrus.Fatal("bind failure:", err)
	}

	logrus.Info("Listing icmp packets...")
	for {
		buf := make([]byte, 1500)
		n, raddr, err := p.recvICMPConn.ReadFrom(buf)
		if err != nil {
			break
		}
		icmpType := buf[0]
		//		logrus.Warn(raddr, "|", icmpType, "|", n)
		if (icmpType == 11 || (icmpType == 3 && buf[1] == 3)) && (n >= 36) { //TTL Exceeded or Port Unreachable
			id := binary.BigEndian.Uint16(buf[12:14])
			//ttl := buf[16]

			dstip := net.IP(buf[24:28])
			srcip := net.IP(buf[20:24])
			srcPort := binary.BigEndian.Uint16(buf[28:30])
			dstPort := binary.BigEndian.Uint16(buf[30:32])

			m := &stats.Metric{
				SrcAddr: srcip.String(),
				DstAddr: dstip.String(),
				SrcPort: srcPort,
				DstPort: dstPort,
				//	TTL:       ttl,
				ID:        id,
				RespAddr:  raddr.String(),
				TimeStamp: time.Now(),
			}
			p.RecvChan <- m
		}
		if atomic.LoadInt32(p.stopSignal) == 1 {
			break
		}
	}
}

func (p *ProbeClient) Start() {

	go p.Recv()

	//open socket
	conn, err := net.ListenPacket("ip4:udp", p.netSrcAddr.String())
	if err != nil {
		logrus.Fatal("open socket failed:", err)
	}
	defer conn.Close()

	Sock, err := ipv4.NewRawConn(conn)
	if err != nil {
		logrus.Fatal("can not create raw socket:", err)
	}

	//ID in IPv4 field as round number.
	id := uint16(1)
	mod := uint16(1 << 15)

	//SendPacket
	for {
		id = (id + uint16(p.MaxTTL) + 1) % mod
		for idx := 0; idx < len(p.netDstAddr); idx++ {
			for pathNum := 0; pathNum < p.MaxPath; pathNum++ {
				pinfo := &stats.ProbeInfo{
					SrcAddr: p.netSrcAddr,
					DstAddr: p.netDstAddr[idx],
					SrcPort: p.SrcPort[pathNum],
					DstPort: p.DstPort[pathNum],
				}

				for ttl := 1; ttl <= int(p.MaxTTL); ttl++ {
					pinfo.TTL = uint8(ttl)
					pinfo.ID = id + uint16(ttl)
					hdr, payload := BuildIPv4UDPkt(pinfo, 0)
					Sock.WriteTo(hdr, payload, nil)
					report := &stats.Metric{
						HostName:  p.Dest[idx],
						SrcAddr:   pinfo.SrcAddr.String(),
						DstAddr:   p.netDstAddr[idx].String(),
						SrcPort:   p.SrcPort[pathNum],
						DstPort:   p.DstPort[pathNum],
						ID:        uint16(hdr.ID),
						TTL:       uint8(ttl),
						TimeStamp: time.Now(),
					}
					p.SendChan <- report
				}
				time.Sleep(p.PacketInterval)
			}
		}
		if atomic.LoadInt32(p.stopSignal) == 1 {
			break
		}
		time.Sleep(p.RoundInterval)
	}
}

func (p *ProbeClient) Stop() {
	atomic.StoreInt32(p.stopSignal, 1)
	p.recvICMPConn.Close()
}
