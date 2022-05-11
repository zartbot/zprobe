package stats

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func std_stats(num []float64, N int) (float64, float64) {
	var sum, mean, sd float64
	for i := 0; i < N; i++ {
		sum += num[i]
	}

	mean = sum / float64(N)
	for j := 0; j < N; j++ {
		sd += math.Pow(num[j]-mean, 2)
	}
	sd = math.Sqrt(sd / float64(N))
	return mean, sd
}
func TestAccuracy(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	var MaxN int = 10000
	var WinSize int = 100
	var eps float64 = 1e-5
	d := make([]float64, MaxN)
	for i := 0; i < MaxN; i++ {
		d[i] = float64(i) + float64(rand.Intn(4000+3*i))
	}

	r := NewRollingStatus(WinSize, 64)
	for idx := 0; idx < WinSize; idx++ {
		r.Update(d[idx])
	}

	for idx := WinSize; idx < MaxN; idx++ {
		start := idx - WinSize

		mean, std := std_stats(d[start:idx], WinSize)

		if math.Abs(r.Mean-mean) > eps {
			t.Errorf("Mean: %v, want %v", r.Mean, mean)
		}
		sdev := math.Sqrt(r.VarSum / float64(WinSize))

		if math.Abs(sdev-std) > eps {
			t.Errorf("Stddev: %v, want %v", sdev, std)
		}
		r.Update(d[idx])
	}

}

func BenchmarkStandardVar(b *testing.B) {
	var WinSize int = 100
	d := make([]float64, b.N)
	for i := 0; i < b.N; i++ {
		d[i] = float64(i) + float64(rand.Intn(4000+3*i))
	}
	for idx := WinSize; idx < b.N; idx++ {
		start := idx - WinSize
		std_stats(d[start:idx], WinSize)
	}
}

func BenchmarkRollingVar(b *testing.B) {
	var WinSize int = 100
	d := make([]float64, b.N)
	for i := 0; i < b.N; i++ {
		d[i] = float64(i) + float64(rand.Intn(4000+3*i))
	}
	r := NewRollingStatus(WinSize, 64)
	for idx := 0; idx < b.N; idx++ {
		r.Update(d[idx])
	}
}
