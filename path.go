package main

import (
	"fmt"
	"strings"
)

type State struct {
	HP                 int16
	ATK                int8
	DEF                int8
	YellowKeys         int8 // 黄钥匙数量
	Money              int16
	DefeatedMonsters   map[int]bool
	CollectedTreasures map[int]bool
	Path               []string
}

type SearchResult struct {
	HP             int16
	ATK            int8
	DEF            int8
	YellowKeys     int8
	Path           []string
	DefeatedCount  int
	CollectedCount int
	Message        string
}

// 寻找最优路径
func findOptimalPath(graph *Graph, startArea, endArea int, initialHP int16, initialATK, initialDEF, initialYellowKeys, requiredATK, requiredDEF, requiredYellowKeys int8) SearchResult {
	// 获取所有怪物和宝物
	allMonsters := []*GlobalMonster{}
	allTreasures := []*GlobalTreasure{}

	for _, area := range graph.Areas {
		// 收集宝物
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

	// 从图的怪物连接中收集怪物信息，避免重复
	for pos, monsterConn := range graph.MonsterConnections {
		allMonsters = append(allMonsters, &GlobalMonster{
			Key:            pos,
			ID:             monsterConn.MonsterID,
			Monster:        monsterConn.Monster,
			Pos:            monsterConn.MonsterPos,
			ConnectedAreas: monsterConn.ConnectedAreas,
		})
	}

	// 计算可访问区域
	getAccessibleAreas := func(defeatedMonsters map[int]bool) map[int]bool {
		accessible := make(map[int]bool)
		accessible[startArea] = true
		queue := []int{startArea}

		for len(queue) > 0 {
			currentArea := queue[0]
			queue = queue[1:]

			// 检查每个怪物
			for monsterIdx, monster := range allMonsters {
				if defeatedMonsters[monsterIdx] { // 怪物已被击败
					// 检查当前区域是否在怪物连接的区域中
					for _, areaID := range monster.ConnectedAreas {
						if areaID == currentArea {
							// 如果当前区域连接到这个怪物，则所有怪物连接的区域都变为可访问
							for _, connectedAreaID := range monster.ConnectedAreas {
								if !accessible[connectedAreaID] {
									accessible[connectedAreaID] = true
									queue = append(queue, connectedAreaID)
								}
							}
							break
						}
					}
				}
			}
		}

		return accessible
	}

	// 计算可收集的宝物
	getCollectibleTreasures := func(accessibleAreas map[int]bool, collectedTreasures map[int]bool) []int {
		collectible := []int{}
		for idx, treasure := range allTreasures {
			if !collectedTreasures[idx] && accessibleAreas[treasure.AreaID] {
				collectible = append(collectible, idx)
			}
		}
		return collectible
	}

	// 状态编码
	encodeState := func(defeatedMonsters map[int]bool, yellowKeys int8) string {
		state := ""
		for i := 0; i < len(allMonsters); i++ {
			if defeatedMonsters[i] {
				state += "1"
			} else {
				state += "0"
			}
		}
		state += fmt.Sprintf("_YK%d", yellowKeys)
		return state
	}

	// 应用宝物效果
	applyTreasures := func(hp int16, atk, def, yellowKey int8, treasureIndices []int) (int16, int8, int8, int8, []string) {
		newHP, newATK, newDEF, newYellowKeys := hp, atk, def, yellowKey
		applied := []string{}

		for _, idx := range treasureIndices {
			treasure := allTreasures[idx]
			applied = append(applied, fmt.Sprintf("%s+%d", treasure.Type, treasure.Value))
			if treasure.Type == treasureHP {
				newHP += int16(treasure.Value)
			} else if treasure.Type == treasureATK {
				newATK += treasure.Value
			} else if treasure.Type == treasureDEF {
				newDEF += treasure.Value
			} else if treasure.Type == treasureYellowKey {
				newYellowKeys += treasure.Value
			}
		}

		return newHP, newATK, newDEF, newYellowKeys, applied
	}

	// DP表
	dp := make(map[string]*State)

	// 初始状态
	initialDefeated := make(map[int]bool)
	initialCollected := make(map[int]bool)
	initialAccessible := getAccessibleAreas(initialDefeated)
	initialCollectible := getCollectibleTreasures(initialAccessible, initialCollected)

	// 收集初始可达的宝物
	newHP, newATK, newDEF, newYellowKeys, applied := applyTreasures(initialHP, initialATK, initialDEF, initialYellowKeys, initialCollectible)
	newInitialCollected := make(map[int]bool)
	for k, v := range initialCollected {
		newInitialCollected[k] = v
	}
	for _, idx := range initialCollectible {
		newInitialCollected[idx] = true
	}

	initialStateKey := encodeState(initialDefeated, newYellowKeys)
	path := []string{fmt.Sprintf("起点区域%d", startArea)}
	if len(applied) > 0 {
		path = append(path, fmt.Sprintf("收集宝物: %s", strings.Join(applied, ", ")))
	}

	dp[initialStateKey] = &State{
		HP:                 newHP,
		ATK:                newATK,
		DEF:                newDEF,
		YellowKeys:         newYellowKeys,
		DefeatedMonsters:   initialDefeated,
		CollectedTreasures: newInitialCollected,
		Path:               path,
	}

	// BFS搜索
	queue := []string{initialStateKey}
	visited := make(map[string]bool)
	visited[initialStateKey] = true

	var bestResult *SearchResult
	iterations := 0
	maxIterations := 50000000

	for len(queue) > 0 && iterations < maxIterations {
		iterations++
		stateKey := queue[0]
		queue = queue[1:]
		state, exists := dp[stateKey]

		if !exists {
			continue
		}

		// 检查是否满足胜利条件
		accessibleAreas := getAccessibleAreas(state.DefeatedMonsters)
		if accessibleAreas[endArea] && state.ATK >= requiredATK && state.DEF >= requiredDEF && state.YellowKeys >= requiredYellowKeys {
			if bestResult == nil || state.HP > bestResult.HP {
				newPath := make([]string, len(state.Path))
				copy(newPath, state.Path)
				newPath = append(newPath, fmt.Sprintf("到达终点区域%d (ATK:%d/%d, DEF:%d/%d, YellowKey:%d/%d)",
					endArea, state.ATK, requiredATK, state.DEF, requiredDEF, state.YellowKeys, requiredYellowKeys))

				bestResult = &SearchResult{
					HP:             state.HP,
					ATK:            state.ATK,
					DEF:            state.DEF,
					YellowKeys:     state.YellowKeys,
					Path:           newPath,
					DefeatedCount:  len(state.DefeatedMonsters),
					CollectedCount: len(state.CollectedTreasures),
				}
			}
		}

		// 尝试击败怪物
		for monsterIdx, monster := range allMonsters {
			if state.DefeatedMonsters[monsterIdx] {
				continue // 已击败
			}

			// 检查怪物是否可达（怪物连接的任一区域可达）
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
			//检查是否有足够的黄钥匙
			if monster.ID == 81 && state.YellowKeys <= 0 {
				continue
			}
			// 计算战斗伤害
			damage := getDamage(state.ATK, state.DEF, monster.ID)
			if damage >= state.HP {
				continue // 无法击败
			}

			newHP := state.HP - damage
			newYellowKeys := state.YellowKeys
			newDefeated := make(map[int]bool)
			for k, v := range state.DefeatedMonsters {
				newDefeated[k] = v
			}
			newDefeated[monsterIdx] = true
			// 是黄门，黄钥匙-1
			if monster.ID == 81 && state.YellowKeys >= 0 {
				newYellowKeys -= 1
			}

			// 计算击败怪物后的新可达区域和可收集宝物
			newAccessible := getAccessibleAreas(newDefeated)
			newCollectible := getCollectibleTreasures(newAccessible, state.CollectedTreasures)

			// 应用新宝物
			finalHP, finalATK, finalDEF, finalYK, treasureApplied := applyTreasures(newHP, state.ATK, state.DEF, newYellowKeys, newCollectible)
			finalCollected := make(map[int]bool)
			for k, v := range state.CollectedTreasures {
				finalCollected[k] = v
			}
			for _, idx := range newCollectible {
				finalCollected[idx] = true
			}

			newStateKey := encodeState(newDefeated, finalYK)

			// 检查是否是更优状态
			existingState, exists := dp[newStateKey]
			if !exists || finalHP > existingState.HP {
				newPath := make([]string, len(state.Path))
				copy(newPath, state.Path)
				newPath = append(newPath, fmt.Sprintf("击败%s@[%d,%d] (伤害:%d, HP:%d)",
					monster.Monster.ID, monster.Pos[0], monster.Pos[1], damage, finalHP))

				if len(treasureApplied) > 0 {
					newPath = append(newPath, fmt.Sprintf("收集宝物: %s", strings.Join(treasureApplied, ", ")))
				}

				dp[newStateKey] = &State{
					HP:                 finalHP,
					ATK:                finalATK,
					DEF:                finalDEF,
					YellowKeys:         finalYK,
					DefeatedMonsters:   newDefeated,
					CollectedTreasures: finalCollected,
					Path:               newPath,
				}

				if !visited[newStateKey] {
					visited[newStateKey] = true
					queue = append(queue, newStateKey)
				}
			}
		}
	}

	if bestResult != nil {
		fmt.Printf("\n=== 找到最优解 ===\n")
		fmt.Printf("最终属性: HP=%d, ATK=%d, DEF=%d\n", bestResult.HP, bestResult.ATK, bestResult.DEF)
		fmt.Printf("\n路径步骤:\n")
		for i, step := range bestResult.Path {
			fmt.Printf("%d. %s\n", i+1, step)
		}
		return *bestResult
	} else {
		message := "无法达到要求的属性并到达终点"
		if maxATK < requiredATK || maxDEF < requiredDEF {
			message = "原因: 宝物不足，无法达到要求属性"
		} else {
			message = "原因: 无法在保持足够HP的情况下到达终点"
		}
		fmt.Println(message)
		return SearchResult{
			HP:      -1,
			Path:    []string{"未找到有效路径"},
			Message: fmt.Sprintf("无法达到要求的属性 (ATK>=%d, DEF>=%d) 并到达终点", requiredATK, requiredDEF),
		}
	}
}
