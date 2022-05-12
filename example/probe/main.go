package main

import (
	"time"

	"github.com/zartbot/zprobe"
	"github.com/zartbot/zprobe/geoip"
	"github.com/zartbot/zprobe/stats"
)

func main() {

	probeName := "zartbot"
	probeList := []string{
		"www.baidu.com",
		"www.tencent.com",
		"cn.taobao.com",
		"www.jd.com",
		"www.weibo.com",
		"www.163.com.cn",
		"www.sohu.com",
		"www.sina.com.cn",
		"www.qq.com",
		"www.youku.com",
		"www.zhihu.com",
		"www.iqiyi.com",
		"www.bilibili.com",
		"www.douban.com",
		"www.sjtu.edu.cn",
		"www.tsinghua.edu.cn",
		"www.mit.edu",
		"www.online.sh.cn",
		"www.cisco.com",
		"www.github.com",
		"www.google.com",
		"www.facebook.com",
		"www.twitter.com",
		"www.amazon.com",
		"www.aws.com",
		"www.netflix.com",
		"www.ebay.com",
		"www.office365.com",
		"www.salesforce.com",
		"video.huawan.com",
		"www.zoom.com",
		"www.webex.com",
		"meeting.tencent.com"}

	maxPath := 4
	maxTTL := 32

	var DelayWinSize int = 32
	var LossWinSize int = 64

	//create SessionDB
	SessionDB := stats.NewSessionDB(probeName, probeList, maxPath, maxTTL, DelayWinSize, LossWinSize)

	p := zprobe.New(probeName, probeList, maxPath, maxTTL, 2*time.Second)
	p.SetPacketInterval(5 * time.Millisecond)
	p.SetRoundInterval(5 * time.Second)

	go p.Start()
	g := geoip.New("../geoip/geoip.mmdb", "../geoip/asn.mmdb")

	go stats.ProcessRecord(p.Report, g, SessionDB, DelayWinSize)
	stats.PrintDB(SessionDB, probeName, probeList, maxPath)

}
