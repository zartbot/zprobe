package main

import (
	"fmt"

	"github.com/zartbot/zprobe"
)

func main() {

	dst := []string{"www.sina.com", "www.taobao.com"}
	p := zprobe.New("zartbot", dst, 4, 32)
	go p.Start()

	for {
		e1 := <-p.Report
		fmt.Printf("%s\n", e1.String())

	}
}
