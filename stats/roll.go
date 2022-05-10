package stats

import (
	"math"
)

type LossBitMap struct {
	index uint64
	data  []uint32
	size  uint64
}

func NewLossBitMap(size uint64) *LossBitMap {
	size = (size + 31) / 32 * 32
	bitmap := &LossBitMap{
		index: 0,
		size:  size,
		data:  make([]uint32, size/32, size/32),
	}
	return bitmap
}

func (b *LossBitMap) Set(pos uint64) bool {
	if pos >= b.size {
		return false
	}
	b.data[pos>>5] |= 1 << (pos & 0x1F)
	return true
}

func (b *LossBitMap) Unset(pos uint64) bool {
	if pos >= b.size {
		return false
	}
	b.data[pos>>5] &= ^(1 << (pos & 0x1F))
	return true
}

func (b *LossBitMap) UpdateLoss() {
	b.Set(b.index)
	b.index = (b.index + 1) % b.size
}

func (b *LossBitMap) UpdatePass() {
	b.Unset(b.index)
	b.index = (b.index + 1) % b.size
}

func swar(i uint32) uint32 {
	// 第一步：计算每2位二进制数中1的个数
	i = (i & 0x55555555) + ((i >> 1) & 0x55555555)
	// 第二步：计算每4位二进制数中1的个数
	i = (i & 0x33333333) + ((i >> 2) & 0x33333333)
	// 第三步：计算每8位二进制数中1的个数
	i = (i & 0x0F0F0F0F) + ((i >> 4) & 0x0F0F0F0F)
	// 第四步：将每8位当做一个int8的整数，然后相加求和
	return (i * 0x01010101) >> 24
}

func (b *LossBitMap) BitCount() uint64 {
	var cnt uint64 = 0
	for i := 0; i < len(b.data); i++ {
		cnt += uint64(swar(b.data[i]))
	}
	return cnt
}

type RollingStatus struct {
	windowSize int
	Mean       float64
	VarSum     float64
	data       []float64
	index      int
	loss       *LossBitMap
}

func NewRollingStatus(delayWinSize int, lossWinSize int) *RollingStatus {
	return &RollingStatus{
		windowSize: delayWinSize,
		Mean:       0,
		VarSum:     0,
		data:       make([]float64, delayWinSize),
		index:      0,
		loss:       NewLossBitMap(uint64(lossWinSize)),
	}
}

func (r *RollingStatus) Update(x_new float64) {
	next_index := (r.index + 1) % r.windowSize
	new_mean := r.Mean + (x_new-r.data[next_index])/float64(r.windowSize)
	r.VarSum = r.VarSum + (x_new-r.Mean)*(x_new-new_mean) -
		(r.data[next_index]-r.Mean)*(r.data[next_index]-new_mean)
	r.data[next_index] = x_new
	r.index = next_index
	r.Mean = new_mean
	r.loss.UpdatePass()
}

func (r *RollingStatus) UpdateLoss() {
	r.loss.UpdateLoss()
}

func (r *RollingStatus) Get() (float64, float64, float64) {
	sd := math.Sqrt(r.VarSum / float64(r.windowSize))
	return r.Mean, sd, float64(r.loss.BitCount()) / float64(r.loss.size)
}
