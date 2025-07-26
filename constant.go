package main

const (
	minATK    = int8(5)
	maxATK    = int8(20)
	minDEF    = int8(5)
	maxDEF    = int8(20)
	maxDamage = int16(9999)

	treasureHP        = 0
	treasureATK       = 1
	treasureDEF       = 2
	treasureYellowKey = 3
	treasureBlueKey   = 4

	maxYellowKey = int8(1<<3 - 1)
	maxBlueKey   = int8(1<<2 - 1)
	// for moving keys to the left
	blueKeyBit   = 62
	yellowKeyBit = 59

	YellowDoorID = 81 // 黄门
	BlueDoorID   = 82 // 蓝门

	maxIterations = 1 << 50
)
