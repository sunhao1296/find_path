package main

// 全局变量
var (
	treasureHP        = 0
	treasureATK       = 1
	treasureDEF       = 2
	treasureYellowKey = 3
	gameMap           = [][]int{
		{1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1},
		{1, 31, 0, 202, 28, 31, 201, 27, 202, 1, 27, 0, 1},
		{1, 203, 1, 1, 1, 206, 1, 1, 31, 204, 0, 1, 1},
		{1, 0, 27, 1, 28, 0, 203, 0, 206, 1, 31, 0, 1},
		{1, 1, 1, 1, 205, 1, 1, 31, 0, 1, 1, 1, 1},
		{1, 0, 204, 0, 0, 27, 1, 0, 203, 27, 1, 27, 1},
		{1, 31, 0, 1, 206, 1, 1, 206, 1, 1, 1, 31, 1},
		{1, 1, 28, 1, 28, 0, 1, 0, 0, 28, 1, 0, 1},
		{1, 31, 202, 1, 1, 203, 1, 1, 203, 1, 1, 204, 1},
		{1, 0, 201, 1, 31, 0, 1, 31, 0, 1, 0, 28, 1},
		{1, 1, 28, 1, 1, 1, 1, 1, 202, 1, 205, 1, 1},
		{1, 31, 0, 0, 0, 205, 0, 201, 0, 27, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1, 21, 1, 1, 1, 1, 1},
	}

	treasureMap = map[int]*Treasure{
		27: {Type: treasureATK, Value: 1},
		28: {Type: treasureDEF, Value: 1},
		31: {Type: treasureHP, Value: 50},
		21: {Type: treasureYellowKey, Value: 1},
	}

	monsterMap = map[int]*Monster{
		201: {HP: 50, ATK: 19, DEF: 1},
		202: {HP: 40, ATK: 22, DEF: 0},
		203: {HP: 35, ATK: 23, DEF: 3},
		204: {HP: 44, ATK: 17, DEF: 2},
		205: {HP: 28, ATK: 25, DEF: 3},
		206: {HP: 33, ATK: 30, DEF: 1},
		81:  {HP: 1}, // 黄门视作怪物
	}

	start = [2]int{11, 6}
	end   = [2]int{0, 7}

	minATK    = int8(8)
	maxATK    = int8(15)
	minDEF    = int8(8)
	maxDEF    = int8(15)
	maxDamage = int16(9999)

	damageCache = make(map[int8]map[int8]map[int]int16)
)
