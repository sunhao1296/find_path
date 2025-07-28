package main

// 位操作函数
func setBit(mask int64, index int) int64 {
	return mask | (1 << index)
}

func hasBit(mask int64, index int) bool {
	return (mask & (1 << index)) != 0
}

func countBits(mask int64) int {
	count := 0
	for mask != 0 {
		count++
		mask &= mask - 1 // 清除最低位的1
	}
	return count
}

// ExtendedBitSet 可扩展的位图结构
type ExtendedBitSet struct {
	bits []uint64
	size int
}

// NewExtendedBitSet 创建新的位图
func NewExtendedBitSet(maxBits int) *ExtendedBitSet {
	size := (maxBits + 63) / 64 // 向上取整
	if size == 0 {
		size = 1
	}
	return &ExtendedBitSet{
		bits: make([]uint64, size),
		size: size,
	}
}

// Set 设置指定位
func (ebs *ExtendedBitSet) Set(pos int) {
	if pos < 0 {
		return
	}
	wordIdx := pos / 64
	bitIdx := pos % 64

	// 自动扩展
	for wordIdx >= len(ebs.bits) {
		ebs.bits = append(ebs.bits, 0)
		ebs.size++
	}

	ebs.bits[wordIdx] |= (1 << bitIdx)
}

// IsSet 检查指定位是否设置
func (ebs *ExtendedBitSet) IsSet(pos int) bool {
	if pos < 0 {
		return false
	}
	wordIdx := pos / 64
	bitIdx := pos % 64

	if wordIdx >= len(ebs.bits) {
		return false
	}

	return (ebs.bits[wordIdx] & (1 << bitIdx)) != 0
}

// Copy 复制位图
func (ebs *ExtendedBitSet) Copy() *ExtendedBitSet {
	newBits := make([]uint64, len(ebs.bits))
	copy(newBits, ebs.bits)
	return &ExtendedBitSet{
		bits: newBits,
		size: ebs.size,
	}
}

// Count 计算设置的位数
func (ebs *ExtendedBitSet) Count() int {
	count := 0
	for _, word := range ebs.bits {
		count += popCount(word)
	}
	return count
}

// popCount 计算一个64位整数中设置的位数
func popCount(x uint64) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// Equal 比较两个位图是否相等
func (ebs *ExtendedBitSet) Equal(other *ExtendedBitSet) bool {
	maxLen := len(ebs.bits)
	if len(other.bits) > maxLen {
		maxLen = len(other.bits)
	}

	for i := 0; i < maxLen; i++ {
		var a, b uint64
		if i < len(ebs.bits) {
			a = ebs.bits[i]
		}
		if i < len(other.bits) {
			b = other.bits[i]
		}
		if a != b {
			return false
		}
	}
	return true
}

// Hash 为位图生成哈希值
func (ebs *ExtendedBitSet) Hash() uint64 {
	var hash uint64 = 0
	for i, word := range ebs.bits {
		hash ^= word * uint64(i*31+1)
	}
	return hash
}

// 辅助函数：检查是否设置了指定位（兼容原有代码）
func hasBitExt(bitset *ExtendedBitSet, pos int) bool {
	return bitset.IsSet(pos)
}

// 辅助函数：设置指定位（兼容原有代码）
func setBitExt(bitset *ExtendedBitSet, pos int) *ExtendedBitSet {
	newBitset := bitset.Copy()
	newBitset.Set(pos)
	return newBitset
}
