package main

// 全局变量
var (
	gameMap = [][]int{
		{31, 1, 27, 0, 203, 0, 28, 81, 0, 202, 0, 81, 27},
		{0, 206, 0, 31, 1, 1, 1, 1, 31, 1, 205, 0, 201},
		{1, 0, 82, 1, 1, 27, 1, 28, 1, 1, 1, 0, 1},
		{28, 202, 0, 27, 1, 203, 0, 203, 0, 1, 0, 206, 0},
		{1, 1, 206, 0, 201, 0, 0, 1, 204, 0, 0, 1, 206},
		{31, 31, 0, 1, 31, 1, 0, 1, 81, 1, 31, 1, 27},
		{1, 81, 1, 1, 1, 1, 206, 31, 206, 1, 0, 1, 31},
		{0, 203, 28, 31, 0, 205, 0, 1, 27, 82, 0, 1, 205},
		{0, 1, 0, 1, 1, 0, 1, 1, 1, 1, 202, 81, 0},
		{202, 1, 82, 28, 202, 0, 201, 0, 31, 1, 0, 1, 203},
		{31, 1, 0, 1, 1, 1, 31, 1, 0, 202, 0, 82, 28},
		{27, 1, 0, 201, 31, 1, 0, 1, 1, 0, 1, 0, 1},
		{0, 204, 0, 1, 31, 203, 0, 1, 31, 201, 0, 201, 28},
	}

	treasureMap = map[int]*Treasure{
		27: {Type: treasureATK, Value: 1},
		28: {Type: treasureDEF, Value: 1},
		31: {Type: treasureHP, Value: 50},
		21: {Type: treasureYellowKey, Value: 1},
		22: {Type: treasureBlueKey, Value: 1},
	}

	monsterMap = map[int]*Monster{
		201: {HP: 48, ATK: 18, DEF: 2},
		202: {HP: 42, ATK: 25, DEF: 1},
		203: {HP: 57, ATK: 16, DEF: 1},
		204: {HP: 44, ATK: 30, DEF: 0},
		205: {HP: 36, ATK: 23, DEF: 4},
		206: {HP: 31, ATK: 33, DEF: 3},
		81:  {HP: 1}, // 黄门视作怪物
		82:  {HP: 1}, // 蓝门视作怪物
	}

	start      = [2]int{11, 6}
	end        = [2]int{0, 6}
	initialAtk = int8(9)    // 初始攻击力
	initialDef = int8(5)    // 初始防御力
	initialHP  = int16(160) // 初始生命值
	initialYK  = int8(2)    // 初始黄钥匙
	initialBK  = int8(1)    // 初始蓝钥匙

	requiredATK        = int8(16) // 需要的攻击力
	requiredDEF        = int8(12) // 需要的防御力
	requiredYellowKeys = int8(0)
	requiredBlueKeys   = int8(0)

	damageCache = make(map[int8]map[int8]map[int]int16)
)
