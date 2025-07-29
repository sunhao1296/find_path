package main

import (
	"container/heap"
	"fmt"
	"sort"
)

type State struct {
	Money             uint8
	ATK               int8  // 1字节
	DEF               int8  // 1字节
	MDEF              uint8 // 1字节 - 新增魔法防御
	YellowKeys        int8  // 1字节
	BlueKeys          int8  // 1字节 - 新增蓝钥匙
	ConsecutiveFights int8  // 连续战斗次数（未提升攻防时）
	FightsSinceStart  int8  // 从开始到现在的战斗次数

	// 新增：购买次数跟踪
	ATKBuys uint8 // 购买攻击的次数
	DEFBuys uint8 // 购买防御的次数

	HP     int16    // 2字节
	Action [2]int16 // 新增：当前动作 [damage, pos编码]
	// 剪枝相关字段

	DefeatedMonsters   int64 // 8字节
	CollectedTreasures int64 // 8字节
	PrevKey            int64 // 新增：前驱状态的key
}

// 优先队列中的状态项
type StateItem struct {
	Key      int64
	Priority int64 // 优先级：越大越优先（血量*1000000 + 金币*1000 - 战斗次数）
	Index    int   // 在堆中的索引
}

// 优先队列实现
type PriorityQueue []*StateItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// 最大堆：优先级高的在前
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*StateItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

// 计算状态优先级
func calculatePriority(hp int16, money uint8, fightsSinceStart int8) int64 {
	// 优先级 = 血量*1000000 + 金币*1000 - 战斗次数
	// 这样可以优先选择血量高、金币多、战斗次数少的状态
	return int64(hp)*1000000 + int64(money)*1000 - int64(fightsSinceStart)
}

// 优化后的主函数 - 使用优先队列
func findOptimalPath(graph *Graph, startHero, requiredHero *HeroItem) SearchResult {
	// 获取所有怪物和宝物
	initialHP, initialATK, initialDEF, initialYellowKeys, initialBlueKeys, startArea := startHero.HP, startHero.ATK, startHero.DEF, startHero.YellowKeys, startHero.BlueKeys, startHero.AreaID
	requiredATK, requiredDEF, requiredMDEF, requiredYellowKeys, requiredBlueKeys, endArea := requiredHero.ATK, requiredHero.DEF, requiredHero.MDEF, requiredHero.YellowKeys, requiredHero.BlueKeys, requiredHero.AreaID
	allMonsters := []*GlobalMonster{}
	allTreasures := []*GlobalTreasure{}

	for _, area := range graph.Areas {
		for idx, treasure := range area.Treasures {
			allTreasures = append(allTreasures, &GlobalTreasure{
				ID:         fmt.Sprintf("%d-%d", area.ID, idx),
				AreaID:     area.ID,
				Type:       treasure.Type,
				Value:      treasure.Value,
				OriginalID: treasure.OriginalID,
			})
		}
	}

	for pos, monsterConn := range graph.MonsterConnections {
		allMonsters = append(allMonsters, &GlobalMonster{
			Key:            pos,
			ID:             monsterConn.MonsterID,
			Monster:        monsterConn.Monster,
			Pos:            monsterConn.MonsterPos,
			ConnectedAreas: monsterConn.ConnectedAreas,
		})
	}

	// 按位置坐标排序怪物，确保处理顺序一致
	sort.Slice(allMonsters, func(i, j int) bool {
		if allMonsters[i].Pos[0] == allMonsters[j].Pos[0] {
			return allMonsters[i].Pos[1] < allMonsters[j].Pos[1]
		}
		return allMonsters[i].Pos[0] < allMonsters[j].Pos[0]
	})

	// 初始化缓存系统
	accessCache := NewAccessibilityCache(allMonsters, 100000)

	// 预计算宝物-区域映射
	treasuresByArea := make(map[int][]int)
	for idx, treasure := range allTreasures {
		treasuresByArea[treasure.AreaID] = append(treasuresByArea[treasure.AreaID], idx)
	}

	// 计算可收集的宝物（优化版）
	getCollectibleTreasuresOptimized := func(accessibleAreas map[int]bool, collectedTreasures int64) []int {
		collectible := []int{}
		for areaID := range accessibleAreas {
			if treasures, exists := treasuresByArea[areaID]; exists {
				for _, treasureIdx := range treasures {
					if !hasBit(collectedTreasures, treasureIdx) {
						collectible = append(collectible, treasureIdx)
					}
				}
			}
		}
		return collectible
	}

	// 应用宝物效果
	applyTreasures := func(hp int16, mdef uint8, atk, def, yellowKey, blueKey int8, treasureIndices []int) (int16, uint8, int8, int8, int8, int8) {
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

	// 修改后的状态编码，包含购买次数信息
	encodeState := func(defeatedMonsters int64, yellowKeys, blueKeys int8, money uint8, atkBuys, defBuys uint8) int64 {
		const (
			yellowKeyBit = 45 // 减少4位为购买次数让出空间
			blueKeyBit   = 48
			moneyBit     = 50
			atkBuysBit   = 56 // 攻击购买次数位置（2位，最多3次）
			defBuysBit   = 58 // 防御购买次数位置（2位，最多3次）
			maxYellowKey = 7
			maxBlueKey   = 3
			maxMoney     = 63
		)

		if yellowKeys > maxYellowKey {
			yellowKeys = maxYellowKey
		}
		if blueKeys > maxBlueKey {
			blueKeys = maxBlueKey
		}
		if money > maxMoney {
			money = maxMoney
		}
		if atkBuys > 3 {
			atkBuys = 3
		}
		if defBuys > 3 {
			defBuys = 3
		}
		if defeatedMonsters > (1<<yellowKeyBit)-1 {
			_ = fmt.Errorf("已击杀怪物数量超过最大限制")
		}

		return (int64(defBuys) << defBuysBit) | (int64(atkBuys) << atkBuysBit) |
			(int64(money) << moneyBit) | (int64(blueKeys) << blueKeyBit) |
			(int64(yellowKeys) << yellowKeyBit) | defeatedMonsters
	}

	// DP表和优先队列
	dp := make(map[int64]*State)
	pq := &PriorityQueue{}
	heap.Init(pq)
	inQueue := make(map[int64]bool) // 跟踪哪些状态在队列中

	// 初始状态
	var initialDefeated int64 = 0
	var initialCollected int64 = 0
	initialAccessible := accessCache.GetAccessibleAreas(initialDefeated, startArea)
	initialCollectible := getCollectibleTreasuresOptimized(initialAccessible, initialCollected)

	initialMDEF := uint8(0)
	newHP, newMDEF, newATK, newDEF, newYellowKeys, newBlueKeys := applyTreasures(initialHP, initialMDEF, initialATK, initialDEF, initialYellowKeys, initialBlueKeys, initialCollectible)
	newInitialCollected := initialCollected
	for _, idx := range initialCollectible {
		newInitialCollected = setBit(newInitialCollected, idx)
	}

	initialStateKey := encodeState(initialDefeated, newYellowKeys, newBlueKeys, startHero.Money, 0, 0)

	initialState := &State{
		HP:                 newHP,
		ATK:                newATK,
		DEF:                newDEF,
		MDEF:               newMDEF,
		Money:              startHero.Money,
		YellowKeys:         newYellowKeys,
		BlueKeys:           newBlueKeys,
		ATKBuys:            0,
		DEFBuys:            0,
		DefeatedMonsters:   initialDefeated,
		CollectedTreasures: newInitialCollected,
		PrevKey:            0,
		Action:             [2]int16{0, 0},
		ConsecutiveFights:  0,
		FightsSinceStart:   0,
	}

	dp[initialStateKey] = initialState
	priority := calculatePriority(newHP, startHero.Money, 0)
	heap.Push(pq, &StateItem{Key: initialStateKey, Priority: priority})
	inQueue[initialStateKey] = true

	// 最优解跟踪
	var bestResult *SearchResult
	var bestKey int64
	var iterations int64
	var prunedCount int64

	// 优先队列搜索（Dijkstra算法变种）
	for pq.Len() > 0 && iterations < maxIterations {
		iterations++

		// 取出优先级最高的状态
		item := heap.Pop(pq).(*StateItem)
		stateKey := item.Key
		delete(inQueue, stateKey)

		state, exists := dp[stateKey]
		if !exists {
			continue
		}

		// 使用缓存获取可达区域
		accessibleAreas := accessCache.GetAccessibleAreas(state.DefeatedMonsters, startArea)

		// 剪枝检查
		if shouldPrune(state, requiredATK, requiredDEF, allMonsters, accessibleAreas) {
			prunedCount++
			continue
		}

		// 检查是否到达终点
		if accessibleAreas[endArea] && state.ATK >= requiredATK && state.DEF >= requiredDEF &&
			state.YellowKeys >= requiredYellowKeys && state.BlueKeys >= requiredBlueKeys && state.MDEF >= requiredMDEF {

			// 更新最优解
			if bestResult == nil || state.HP > bestResult.HP ||
				(state.HP == bestResult.HP && state.Money > bestResult.Money) {

				bestKey = stateKey
				bestResult = &SearchResult{
					HP:             state.HP,
					ATK:            state.ATK,
					DEF:            state.DEF,
					MDEF:           state.MDEF,
					Money:          state.Money,
					YellowKeys:     state.YellowKeys,
					BlueKeys:       state.BlueKeys,
					DefeatedCount:  countBits(state.DefeatedMonsters),
					CollectedCount: countBits(state.CollectedTreasures),
				}
			}

			// 找到一个解后可以继续搜索更优解，或者直接返回
			// 如果想要绝对最优解，继续搜索；如果想要第一个可行解，可以break
			continue
		}

		// 尝试击败怪物
		for monsterIdx, monster := range allMonsters {
			if hasBit(state.DefeatedMonsters, monsterIdx) {
				continue
			}

			// 检查怪物是否可达
			canReachMonster := false
			for _, areaID := range monster.ConnectedAreas {
				if accessibleAreas[areaID] {
					canReachMonster = true
					break
				}
			}

			if !canReachMonster {
				continue
			}

			// 检查钥匙需求
			if monster.ID == YellowDoorID && state.YellowKeys <= 0 {
				continue
			}
			if monster.ID == BlueDoorID && state.BlueKeys <= 0 {
				continue
			}

			damage := getDamage(state.ATK, state.DEF, monster.ID)
			if damage >= state.HP {
				continue
			}

			newHP := state.HP - damage
			newMoney := state.Money + monster.Monster.Money
			newYellowKeys := state.YellowKeys
			newBlueKeys := state.BlueKeys
			newDefeated := setBit(state.DefeatedMonsters, monsterIdx)

			// 消耗钥匙
			if monster.ID == YellowDoorID {
				newYellowKeys -= 1
			}
			if monster.ID == BlueDoorID {
				newBlueKeys -= 1
			}

			// 使用增量更新获取新的可达区域
			newAccessible := accessCache.GetAccessibleAreasIncremental(
				state.DefeatedMonsters, monsterIdx, startArea, accessibleAreas)

			newCollectible := getCollectibleTreasuresOptimized(newAccessible, state.CollectedTreasures)

			finalHP, finalMDEF, finalATK, finalDEF, finalYK, finalBK := applyTreasures(newHP, state.MDEF, state.ATK, state.DEF, newYellowKeys, newBlueKeys, newCollectible)
			finalCollected := state.CollectedTreasures
			for _, idx := range newCollectible {
				finalCollected = setBit(finalCollected, idx)
			}

			// 计算新的剪枝状态
			oldAtkDef := state.ATK + state.DEF
			newAtkDef := finalATK + finalDEF
			newConsecutiveFights := state.ConsecutiveFights
			if newAtkDef > oldAtkDef {
				newConsecutiveFights = 0
			} else if damage > 0 {
				newConsecutiveFights++
			}

			newStateKey := encodeState(newDefeated, finalYK, finalBK, newMoney, state.ATKBuys, state.DEFBuys)

			// 检查是否需要更新状态
			existingState, exists := dp[newStateKey]
			shouldUpdate := false

			if !exists {
				shouldUpdate = true
			} else {
				// 比较状态优劣：优先血量，其次金币，最后战斗次数
				newPriority := calculatePriority(finalHP, newMoney, state.FightsSinceStart+1)
				oldPriority := calculatePriority(existingState.HP, existingState.Money, existingState.FightsSinceStart)
				shouldUpdate = newPriority > oldPriority
			}

			if shouldUpdate {
				newState := &State{
					HP:                 finalHP,
					ATK:                finalATK,
					DEF:                finalDEF,
					MDEF:               finalMDEF,
					Money:              newMoney,
					YellowKeys:         finalYK,
					BlueKeys:           finalBK,
					ATKBuys:            state.ATKBuys,
					DEFBuys:            state.DEFBuys,
					DefeatedMonsters:   newDefeated,
					CollectedTreasures: finalCollected,
					PrevKey:            stateKey,
					Action:             [2]int16{int16(damage), int16(monster.Pos[0]<<8 + monster.Pos[1])},
					ConsecutiveFights:  newConsecutiveFights,
					FightsSinceStart:   state.FightsSinceStart + 1,
				}
				if damage == 0 {
					newState.FightsSinceStart = state.FightsSinceStart
				}
				dp[newStateKey] = newState

				// 加入优先队列（如果不在队列中）
				if !inQueue[newStateKey] {
					priority := calculatePriority(finalHP, newMoney, state.FightsSinceStart+1)
					heap.Push(pq, &StateItem{Key: newStateKey, Priority: priority})
					inQueue[newStateKey] = true
				}
			}

			// 修改后的购买逻辑 - 同时考虑购买ATK和DEF
			if newMoney >= 40 {
				// 购买ATK
				if state.ATKBuys < 3 { // 限制购买次数
					buyMoneyATK := newMoney - 40
					buyATK := finalATK + 1
					newATKBuys := state.ATKBuys + 1
					buyStateKeyATK := encodeState(newDefeated, finalYK, finalBK, buyMoneyATK, newATKBuys, state.DEFBuys)

					existingBuyStateATK, buyExistsATK := dp[buyStateKeyATK]
					shouldUpdateBuyATK := false

					if !buyExistsATK {
						shouldUpdateBuyATK = true
					} else {
						newPriorityATK := calculatePriority(finalHP, buyMoneyATK, state.FightsSinceStart+1)
						oldPriorityATK := calculatePriority(existingBuyStateATK.HP, existingBuyStateATK.Money, existingBuyStateATK.FightsSinceStart)
						shouldUpdateBuyATK = newPriorityATK > oldPriorityATK
					}

					if shouldUpdateBuyATK {
						buyStateATK := &State{
							HP:                 finalHP,
							ATK:                buyATK,
							DEF:                finalDEF,
							MDEF:               finalMDEF,
							Money:              buyMoneyATK,
							YellowKeys:         finalYK,
							BlueKeys:           finalBK,
							ATKBuys:            newATKBuys,
							DEFBuys:            state.DEFBuys,
							DefeatedMonsters:   newDefeated,
							CollectedTreasures: finalCollected,
							PrevKey:            newStateKey,
							Action:             [2]int16{-1, -1}, // 购买ATK
							ConsecutiveFights:  0,
							FightsSinceStart:   state.FightsSinceStart + 1,
						}

						dp[buyStateKeyATK] = buyStateATK

						if !inQueue[buyStateKeyATK] {
							buyPriorityATK := calculatePriority(finalHP, buyMoneyATK, state.FightsSinceStart+1)
							heap.Push(pq, &StateItem{Key: buyStateKeyATK, Priority: buyPriorityATK})
							inQueue[buyStateKeyATK] = true
						}
					}
				}

				// 购买DEF
				if state.DEFBuys < 3 { // 限制购买次数
					buyMoneyDEF := newMoney - 40
					buyDEF := finalDEF + 1
					newDEFBuys := state.DEFBuys + 1
					buyStateKeyDEF := encodeState(newDefeated, finalYK, finalBK, buyMoneyDEF, state.ATKBuys, newDEFBuys)

					existingBuyStateDEF, buyExistsDEF := dp[buyStateKeyDEF]
					shouldUpdateBuyDEF := false

					if !buyExistsDEF {
						shouldUpdateBuyDEF = true
					} else {
						newPriorityDEF := calculatePriority(finalHP, buyMoneyDEF, state.FightsSinceStart+1)
						oldPriorityDEF := calculatePriority(existingBuyStateDEF.HP, existingBuyStateDEF.Money, existingBuyStateDEF.FightsSinceStart)
						shouldUpdateBuyDEF = newPriorityDEF > oldPriorityDEF
					}

					if shouldUpdateBuyDEF {
						buyStateDEF := &State{
							HP:                 finalHP,
							ATK:                finalATK,
							DEF:                buyDEF,
							MDEF:               finalMDEF,
							Money:              buyMoneyDEF,
							YellowKeys:         finalYK,
							BlueKeys:           finalBK,
							ATKBuys:            state.ATKBuys,
							DEFBuys:            newDEFBuys,
							DefeatedMonsters:   newDefeated,
							CollectedTreasures: finalCollected,
							PrevKey:            newStateKey,
							Action:             [2]int16{-2, -2}, // 购买DEF
							ConsecutiveFights:  0,
							FightsSinceStart:   state.FightsSinceStart + 1,
						}

						dp[buyStateKeyDEF] = buyStateDEF

						if !inQueue[buyStateKeyDEF] {
							buyPriorityDEF := calculatePriority(finalHP, buyMoneyDEF, state.FightsSinceStart+1)
							heap.Push(pq, &StateItem{Key: buyStateKeyDEF, Priority: buyPriorityDEF})
							inQueue[buyStateKeyDEF] = true
						}
					}
				}
			}
		}
	}

	if iterations >= maxIterations {
		fmt.Println("max iterations reached")
	}

	if bestResult != nil {
		bestResult.Path = reconstructPath(dp, bestKey)
		return *bestResult
	} else {
		return SearchResult{
			HP:   -1,
			Path: []int16{},
		}
	}
}
