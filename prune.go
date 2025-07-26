package main

// 剪枝检查函数
func shouldPrune(state *State, requiredATK, requiredDEF int8, allMonsters []*GlobalMonster, accessibleAreas map[int]bool) bool {
	currentAtkDef := state.ATK + state.DEF
	atkDefImprovement := currentAtkDef - initialAtk - initialDef
	// 剪枝策略1: 如果打了5只怪之后攻防和比初始攻防和只高2点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 6 {
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
	if state.FightsSinceStart >= 10 {
		if atkDefImprovement <= 4 {
			return true
		}
	}

	// 剪枝策略1: 如果打了10只怪之后攻防和比初始攻防和只高5点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 15 {
		if atkDefImprovement <= 7 {
			return true
		}
	}

	// 剪枝策略1: 如果打了10只怪之后攻防和比初始攻防和只高5点或更少，则停止扩展该路线
	if state.FightsSinceStart >= 20 {
		if atkDefImprovement <= 9 {
			return true
		}
	}

	return false
}
