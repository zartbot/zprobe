package main

import (
	"fmt"

	"github.com/zartbot/zprobe/stats"
)

func main() {

	s := stats.NewRollingStatus(32, 64)

	for i := 0; i < 1000; i++ {
		s.UpdateLoss()

		d, j, l := s.Get()
		fmt.Printf("%10d  D:%10.2f J:%10.2f L:%10.2f\n", i, d, j, l)
	}
}
