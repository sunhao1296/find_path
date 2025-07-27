package main

// 战斗相关常量
const (
	minATK    = int8(5)
	maxATK    = int8(20)
	minDEF    = int8(5)
	maxDEF    = int8(20)
	maxDamage = int16(9999)
)

// 物品相关常量
const (
	treasureHP        = 0
	treasureATK       = 1
	treasureDEF       = 2
	treasureYellowKey = 3
	treasureBlueKey   = 4

	maxYellowKey = int8(1<<3 - 1)
	maxBlueKey   = int8(1<<2 - 1)
)

// 位操作常量
const (
	// 钥匙在位掩码中的位置
	blueKeyBit   = 62
	yellowKeyBit = 59
)

// 地图元素常量
const (
	YellowDoorID = 81 // 黄门
	BlueDoorID   = 82 // 蓝门
)

// 算法常量
const (
	maxIterations = 1 << 50
)