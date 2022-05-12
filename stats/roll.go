package stats

import (
	"math"
)

var bitsInByte = [256]uint8{
	0, 1, 1, 2, 1, 2, 2, 3, 1, 2, 2, 3, 2, 3, 3, 4,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	4, 5, 5, 6, 5, 6, 6, 7, 5, 6, 6, 7, 6, 7, 7, 8}

type LossBitMap struct {
	index        uint64
	data         []uint32
	size         uint64
	weightedLoss uint8
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

func (b *LossBitMap) UpdateWeightedLoss() {
	b.Set(b.index)
	b.index = (b.index + 1) % b.size
	b.weightedLoss = b.weightedLoss<<1 + 1
}

func (b *LossBitMap) UpdateWeightedPass() {
	b.Unset(b.index)
	b.index = (b.index + 1) % b.size
	b.weightedLoss = b.weightedLoss << 1
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

func (r *RollingStatus) UpdateLatency(xN float64) {
	//TODO: add new field for weighted average ?
	idxN := (r.index + 1) % r.windowSize
	x0 := r.data[idxN]
	meanN := r.Mean + (xN-x0)/float64(r.windowSize)
	r.VarSum = r.VarSum + (xN-x0)*(xN-meanN+x0-r.Mean)
	r.data[idxN] = xN
	r.index = idxN
	r.Mean = meanN
}

func (r *RollingStatus) Update(xN float64) {
	r.UpdateLatency(xN)
	r.loss.UpdatePass()
}

func (r *RollingStatus) UpdateLoss() {
	r.loss.UpdateLoss()
}

func (r *RollingStatus) Get() (float64, float64, float64) {
	sd := math.Sqrt(r.VarSum / float64(r.windowSize))
	return r.Mean, sd, float64(r.loss.BitCount()) / float64(r.loss.size)
}

func (r *RollingStatus) UpdateWeighted(xN float64) {
	r.UpdateLatency(xN)
	r.loss.UpdateWeightedPass()
}

func (r *RollingStatus) UpdateWeightedLoss() {
	r.loss.UpdateWeightedLoss()
}

func (r *RollingStatus) GetWeighted(N uint64) (float64, float64, float64) {
	sd := math.Sqrt(r.VarSum / float64(r.windowSize))
	lossCnt := r.loss.BitCount() +
		N*uint64(bitsInByte[r.loss.weightedLoss])

	return r.Mean, sd, float64(lossCnt) / float64(r.loss.size+N<<3)
}
