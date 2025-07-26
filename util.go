package main

import "fmt"

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

// 输出路径函数（回溯 reconstruct）
func printPath(path []int16) {
	fmt.Printf("\n路径步骤:\n")
	if len(path) == 0 {
		fmt.Println("无战斗记录")
		return
	}
	for i := 0; i < len(path); i += 2 {
		damage := path[i]
		pos := path[i+1]
		fmt.Printf("%d. 战斗损失%d血, 战斗at %d, %d\n", i/2+1, damage, pos>>8, pos%(1<<8))
	}
}
