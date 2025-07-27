package main

import (
	"fmt"
	"sort"
)

type State struct {
	HP         int16 // 2字节
	ATK        int8  // 1字节
	DEF        int8  // 1字节
	YellowKeys int8  // 1字节
	BlueKeys   int8  // 1字节 - 新增蓝钥匙
	// 剪枝相关字段
	ConsecutiveFights int8 // 连续战斗次数（未提升攻防时）
	FightsSinceStart  int8 // 从开始到现在的战斗次数

	Action             [2]int16 // 新增：当前动作 [damage, pos编码]
	DefeatedMonsters   int64    // 8字节
	CollectedTreasures int64    // 8字节
	PrevKey            int64    // 新增：前驱状态的key
}

type SearchResult struct {
	HP             int16
	ATK            int8
	DEF            int8
	YellowKeys     int8
	BlueKeys       int8    // 新增蓝钥匙
	Path           []int16 // 存储每次战斗损失的血量
	DefeatedCount  int
	CollectedCount int
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

type HeroItem struct {
	AreaID     int
	HP         int16
	ATK        int8
	DEF        int8
	YellowKeys int8
	BlueKeys   int8
}

// reconstructPath: 回溯生成完整路径
func reconstructPath(dp map[int64]*State, endKey int64) []int16 {
	path := []int16{}
	for key := endKey; key != 0; {
		state := dp[key]
		if state == nil || (state.PrevKey == 0 && (state.Action[0] == 0 && state.Action[1] == 0)) {
			break
		}
		path = append([]int16{state.Action[0], state.Action[1]}, path...)
		key = state.PrevKey
	}
	return path
}

// 优化后的主函数
func findOptimalPath(graph *Graph, startHero, requiredHero *HeroItem) SearchResult {
	// 获取所有怪物和宝物
	initialHP, initialATK, initialDEF, initialYellowKeys, initialBlueKeys, startArea := startHero.HP, startHero.ATK, startHero.DEF, startHero.YellowKeys, startHero.BlueKeys, startHero.AreaID
	requiredATK, requiredDEF, requiredYellowKeys, requiredBlueKeys, endArea := requiredHero.ATK, requiredHero.DEF, requiredHero.YellowKeys, requiredHero.BlueKeys, requiredHero.AreaID
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
	accessCache := NewAccessibilityCache(allMonsters, 100000) // 缓存最近100000个状态

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

	// 应用宝物效果（更新支持蓝钥匙）
	applyTreasures := func(hp int16, atk, def, yellowKey, blueKey int8, treasureIndices []int) (int16, int8, int8, int8, int8) {
		newHP, newATK, newDEF, newYellowKeys, newBlueKeys := hp, atk, def, yellowKey, blueKey
		for _, idx := range treasureIndices {
			treasure := allTreasures[idx]
			if treasure.Type == treasureHP {
				newHP += int16(treasure.Value)
			} else if treasure.Type == treasureATK {
				newATK += treasure.Value
			} else if treasure.Type == treasureDEF {
				newDEF += treasure.Value
			} else if treasure.Type == treasureYellowKey {
				newYellowKeys += treasure.Value
			} else if treasure.Type == treasureBlueKey {
				newBlueKeys += treasure.Value
			}
		}
		return newHP, newATK, newDEF, newYellowKeys, newBlueKeys
	}

	// 状态编码（修改支持蓝钥匙）
	// 位分配：低59位为怪物状态，第59-61位为黄钥匙(3位，支持0-7)，第62-63位为蓝钥匙(2位，支持0-3)
	encodeState := func(defeatedMonsters int64, yellowKeys, blueKeys int8) int64 {
		if yellowKeys > maxYellowKey {
			_ = fmt.Errorf("黄钥匙数量超过最大限制，已调整为7")
		}
		if blueKeys > maxBlueKey {
			_ = fmt.Errorf("蓝钥匙数量超过最大限制，已调整为3")
		}
		if defeatedMonsters > (1<<yellowKeyBit)-1 {
			_ = fmt.Errorf("已击杀怪物数量超过最大限制")
		}
		return (int64(blueKeys) << blueKeyBit) | (int64(yellowKeys) << yellowKeyBit) | defeatedMonsters
	}

	// DP表
	dp := make(map[int64]*State)

	// 初始状态
	var initialDefeated int64 = 0
	var initialCollected int64 = 0
	initialAccessible := accessCache.GetAccessibleAreas(initialDefeated, startArea)
	initialCollectible := getCollectibleTreasuresOptimized(initialAccessible, initialCollected)

	newHP, newATK, newDEF, newYellowKeys, newBlueKeys := applyTreasures(initialHP, initialATK, initialDEF, initialYellowKeys, initialBlueKeys, initialCollectible)
	newInitialCollected := initialCollected
	for _, idx := range initialCollectible {
		newInitialCollected = setBit(newInitialCollected, idx)
	}

	initialStateKey := encodeState(initialDefeated, newYellowKeys, newBlueKeys)

	dp[initialStateKey] = &State{
		HP:                 newHP,
		ATK:                newATK,
		DEF:                newDEF,
		YellowKeys:         newYellowKeys,
		BlueKeys:           newBlueKeys,
		DefeatedMonsters:   initialDefeated,
		CollectedTreasures: newInitialCollected,
		PrevKey:            0,
		Action:             [2]int16{0, 0},
		ConsecutiveFights:  0,
		FightsSinceStart:   0,
	}

	// BFS搜索（分阶段：第一阶段只扩展状态，第二阶段统一选最优解）
	queue := []int64{initialStateKey}
	visited := make(map[int64]bool)
	visited[initialStateKey] = true

	var candidateKeys []int64 // 阶段一收集所有满足终点条件的状态key
	var iterations int64
	var prunedCount int64 // 统计剪枝次数

	for len(queue) > 0 && iterations < maxIterations {
		iterations++
		stateKey := queue[0]
		queue = queue[1:]
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

		// 阶段一：只收集满足终点条件的状态，不直接更新最优解
		if accessibleAreas[endArea] && state.ATK >= requiredATK && state.DEF >= requiredDEF &&
			state.YellowKeys >= requiredYellowKeys && state.BlueKeys >= requiredBlueKeys {
			candidateKeys = append(candidateKeys, stateKey)
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
			newYellowKeys := state.YellowKeys
			newBlueKeys := state.BlueKeys
			newDefeated := setBit(state.DefeatedMonsters, monsterIdx)

			// 消耗钥匙
			if monster.ID == YellowDoorID {
				newYellowKeys -= 1
			}
			if monster.ID == BlueDoorID {
				newBlueKeys -= 1 // 确保蓝钥匙消耗
			}

			// 使用增量更新获取新的可达区域
			newAccessible := accessCache.GetAccessibleAreasIncremental(
				state.DefeatedMonsters, monsterIdx, startArea, accessibleAreas)

			newCollectible := getCollectibleTreasuresOptimized(newAccessible, state.CollectedTreasures)

			finalHP, finalATK, finalDEF, finalYK, finalBK := applyTreasures(newHP, state.ATK, state.DEF, newYellowKeys, newBlueKeys, newCollectible)
			finalCollected := state.CollectedTreasures
			for _, idx := range newCollectible {
				finalCollected = setBit(finalCollected, idx)
			}

			// 计算新的剪枝状态
			oldAtkDef := state.ATK + state.DEF
			newAtkDef := finalATK + finalDEF
			newConsecutiveFights := state.ConsecutiveFights
			if newAtkDef > oldAtkDef {
				newConsecutiveFights = 0 // 攻防提升，重置连续计数
			} else {
				newConsecutiveFights++ // 没有提升，增加连续计数
			}

			newStateKey := encodeState(newDefeated, finalYK, finalBK)

			existingState, exists := dp[newStateKey]
			if !exists || finalHP > existingState.HP {
				dp[newStateKey] = &State{
					HP:                 finalHP,
					ATK:                finalATK,
					DEF:                finalDEF,
					YellowKeys:         finalYK,
					BlueKeys:           finalBK,
					DefeatedMonsters:   newDefeated,
					CollectedTreasures: finalCollected,
					PrevKey:            stateKey,
					Action:             [2]int16{int16(damage), int16(monster.Pos[0]<<8 + monster.Pos[1])},
					ConsecutiveFights:  newConsecutiveFights,
					FightsSinceStart:   state.FightsSinceStart + 1,
				}

				if !visited[newStateKey] {
					visited[newStateKey] = true
					queue = append(queue, newStateKey)
				}
			}
		}
	}

	// 阶段二：统一在所有满足终点条件的状态中选取最优解
	var bestResult *SearchResult
	var bestKey int64
	for _, key := range candidateKeys {
		state := dp[key]
		if state == nil {
			continue
		}
		if bestResult == nil || state.HP > bestResult.HP {
			bestKey = key
			bestResult = &SearchResult{
				HP:             state.HP,
				ATK:            state.ATK,
				DEF:            state.DEF,
				YellowKeys:     state.YellowKeys,
				BlueKeys:       state.BlueKeys,
				DefeatedCount:  countBits(state.DefeatedMonsters),
				CollectedCount: countBits(state.CollectedTreasures),
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
		// 修复：找不到最优解时，不回溯路径，直接返回空路径
		return SearchResult{
			HP:   -1,
			Path: []int16{},
		}
	}
}
