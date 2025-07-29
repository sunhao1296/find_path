package main

import (
	"math"
)

// 初始化伤害缓存
func initDamageCache() {
	for atk := minATK; atk <= maxATK; atk++ {
		damageCache[atk] = make(map[int8]map[int]int16)
		for def := minDEF; def <= maxDEF; def++ {
			damageCache[atk][def] = make(map[int]int16)
			for monsterID := range monsterMap {
				monster := monsterMap[monsterID]
				playerDamage := atk - monster.DEF

				var damage int16
				if playerDamage <= 0 {
					damage = maxDamage
				} else {
					monsterDamage := int16(math.Max(0, float64(monster.ATK-def)))
					rounds := int16(math.Ceil(float64(monster.HP)/float64(playerDamage))) - 1
					damage = rounds * monsterDamage
				}

				damageCache[atk][def][monsterID] = damage
			}
		}
	}
}

// 获取预计算的伤害值
func getDamage(playerATK, playerDEF int8, monsterID int) int16 {
	return damageCache[playerATK][playerDEF][monsterID]
}

// 剪枝检查函数
func shouldPrune(state *State, requiredATK, requiredDEF int8, allMonsters []*GlobalMonster, accessibleAreas map[int]bool) bool {
	currentAtkDef := state.ATK + state.DEF
	atkDefImprovement := currentAtkDef - initialAtk - initialDef

	// 剪枝策略1
	if state.FightsSinceStart >= 4 {
		if atkDefImprovement == 0 {
			return true
		}
	}

	// 剪枝策略1
	if state.FightsSinceStart >= 6 {
		if atkDefImprovement <= 1 {
			return true
		}
	}

	// 剪枝策略2
	if state.FightsSinceStart >= 7 {
		if atkDefImprovement <= 2 {
			return true
		}
	}

	if state.Money > 45 {
		return true
	}

	// 剪枝策略3
	// 注意：只计算非零伤害的怪物
	if state.ConsecutiveFights >= 5 && (state.ATK < requiredATK-2 || state.DEF < requiredDEF-2) {
		return true
	}

	// 剪枝策略1: 如果打了10只怪之后攻防和比初始攻防和只高5点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 11 {
		if atkDefImprovement <= 4 {
			return true
		}
	}

	// 剪枝策略1: 如果打了10只怪之后攻防和比初始攻防和只高5点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 16 {
		if atkDefImprovement <= 7 {
			return true
		}
	}

	// 剪枝策略1: 如果打了10只怪之后攻防和比初始攻防和只高5点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 21 {
		if atkDefImprovement <= 9 {
			return true
		}
	}

	// 剪枝策略1: 如果打了10只怪之后攻防和比初始攻防和只高5点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 27 {
		if atkDefImprovement <= 12 {
			return true
		}
	}

	return false
}
