package pio

import (
	"bytes"
	"encoding/binary"

	"github.com/sirupsen/logrus"
	"github.com/zartbot/zprobe/stats"
	"golang.org/x/net/ipv4"
)

type UDPHeader struct {
	Src    uint16
	Dst    uint16
	Length uint16
	Chksum uint16
}

//checksum function
func checkSum(buf []byte) uint16 {
	sum := uint32(0)

	for ; len(buf) >= 2; buf = buf[2:] {
		sum += uint32(buf[0])<<8 | uint32(buf[1])
	}
	if len(buf) > 0 {
		sum += uint32(buf[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)
	/*
	 * From RFC 768:
	 * If the computed checksum is zero, it is transmitted as all ones (the
	 * equivalent in one's complement arithmetic). An all zero transmitted
	 * checksum value means that the transmitter generated no checksum (for
	 * debugging or for higher level protocols that don't care).
	 */
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

func (u *UDPHeader) checksum(ip *ipv4.Header, payload []byte) {
	var pseudoHeader []byte

	pseudoHeader = append(pseudoHeader, ip.Src.To4()...)
	pseudoHeader = append(pseudoHeader, ip.Dst.To4()...)
	pseudoHeader = append(pseudoHeader, []byte{
		0,
		17,
		0, byte(u.Length),
	}...)

	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, pseudoHeader)
	binary.Write(&b, binary.BigEndian, u)
	binary.Write(&b, binary.BigEndian, &payload)
	u.Chksum = checkSum(b.Bytes())
}

func BuildIPv4UDPkt(p *stats.ProbeInfo, tos int) (*ipv4.Header, []byte) {
	iph := &ipv4.Header{
		Version:  ipv4.Version,
		TOS:      tos,
		Len:      ipv4.HeaderLen,
		TotalLen: 60,
		ID:       int(p.ID),
		Flags:    0,
		FragOff:  0,
		TTL:      int(p.TTL),
		Protocol: 17,
		Checksum: 0,
		Src:      p.SrcAddr,
		Dst:      p.DstAddr,
	}

	h, err := iph.Marshal()
	if err != nil {
		logrus.Fatal("Failed build ip header:", err)
	}
	iph.Checksum = int(checkSum(h))

	udp := UDPHeader{
		Src: p.SrcPort,
		Dst: p.DstPort,
	}

	payload := make([]byte, 32)
	for i := 0; i < 32; i++ {
		payload[i] = uint8(i + 64)
	}
	udp.Length = uint16(len(payload) + 8)
	udp.checksum(iph, payload)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, &udp)
	binary.Write(&buf, binary.BigEndian, &payload)
	return iph, buf.Bytes()
}
