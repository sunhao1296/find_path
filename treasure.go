package main

type Treasure struct {
	Type  int
	Value int8 // 1 byte
}

type TreasureItem struct {
	ID         int
	Type       int
	Value      int8
	OriginalID int
}

type GlobalTreasure struct {
	ID         string
	AreaID     int
	Type       int
	Value      int8
	OriginalID int
}

// 应用宝物效果（更新支持蓝钥匙）
func applyTreasures(allTreasures []*GlobalTreasure, hp int16, mdef uint8, atk, def, yellowKey, blueKey int8, treasureIndices []int) (int16, uint8, int8, int8, int8, int8) {
	newHP, newATK, newDEF, newMDEF, newYellowKeys, newBlueKeys := hp, atk, def, mdef, yellowKey, blueKey
	for _, idx := range treasureIndices {
		treasure := allTreasures[idx]
		switch treasure.Type {
		case treasureDEF:
			newDEF += treasure.Value
		case treasureATK:
			newATK += treasure.Value
		case treasureHP:
			newHP += int16(treasure.Value)
		case treasureYellowKey:
			newYellowKeys += treasure.Value
		case treasureBlueKey:
			newBlueKeys += treasure.Value
		case treasureMDEF:
			newMDEF += uint8(treasure.Value)
		}
	}
	return newHP, newMDEF, newATK, newDEF, newYellowKeys, newBlueKeys
}
