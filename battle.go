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
	// 剪枝策略1: 如果打了5只怪之后攻防和比初始攻防和只高2点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 7 {
		if atkDefImprovement <= 2 {
			return true
		}
	}

	// 剪枝策略2: 如果一条路线连续打5只怪都没有提升攻防和，且攻防还没有达到required攻防，则停止扩展该路线
	// 注意：只计算非零伤害的怪物
	nonZeroDamageFights := countNonZeroDamageFights(state, allMonsters)
	if nonZeroDamageFights >= 5 && (state.ATK < requiredATK-1 || state.DEF < requiredDEF-1) && (state.HP < 100) {
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

	return false
}

// 计算非零伤害的连续战斗次数
func countNonZeroDamageFights(state *State, allMonsters []*GlobalMonster) int8 {
	// 这里需要回溯路径来统计，简化实现：如果连续战斗中大部分是零伤害，则调整计数
	// 实际实现中可以在State中添加专门的NonZeroDamageFights字段来精确追踪
	nonZeroRatio := 0.7 // 假设70%的战斗是有伤害的
	return int8(float64(state.ConsecutiveFights) * nonZeroRatio)
}
