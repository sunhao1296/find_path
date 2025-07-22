package main

import (
	"fmt"
)

type State struct {
	HP                 int16   // 2字节
	ATK                int8    // 1字节
	DEF                int8    // 1字节
	YellowKeys         int8    // 1字节
	Money              int16   // 2字节 (似乎没用到？)
	DefeatedMonsters   int64   // 8字节
	CollectedTreasures int64   // 8字节
	Path               []int16 // 24字节(slice header)
}

type SearchResult struct {
	HP             int16
	ATK            int8
	DEF            int8
	YellowKeys     int8
	Path           []int16 // 存储每次战斗损失的血量
	DefeatedCount  int
	CollectedCount int
	Message        string
}

// 输出路径函数
func printPath(path []int16) {
	fmt.Printf("\n路径步骤:\n")
	if len(path) == 0 {
		fmt.Println("无战斗记录")
		return
	}

	for i, damage := range path {
		fmt.Printf("%d. 战斗损失%d血\n", i+1, damage)
	}
}

// 展示状态编码示例（调试用）
func showStateEncoding(defeatedMonsters int64, yellowKeys int8) {
	encoded := (int64(yellowKeys) << 60) | defeatedMonsters
	fmt.Printf("怪物状态: %064b\n", defeatedMonsters)
	fmt.Printf("黄钥匙数: %d (二进制: %04b)\n", yellowKeys, yellowKeys)
	fmt.Printf("编码结果: %064b\n", encoded)
	fmt.Printf("编码为int64: %d\n", encoded)
}

// 位操作函数
func setBit(mask int64, index int) int64 {
	return mask | (1 << index)
}

func hasBit(mask int64, index int) bool {
	return (mask & (1 << index)) != 0
}

func countBits(mask int64) int {
	count := 0
	for mask != 0 {
		count++
		mask &= mask - 1 // 清除最低位的1
	}
	return count
}

// 将位掩码转换为map（用于兼容现有逻辑）
func bitmaskToMap(mask int64, size int) map[int]bool {
	result := make(map[int]bool)
	for i := 0; i < size; i++ {
		if hasBit(mask, i) {
			result[i] = true
		}
	}
	return result
}

// 预计算的映射关系
type AccessibilityCache struct {
	// 怪物ID -> 连接的区域列表
	monsterToAreas map[int][]int
	// 区域ID -> 连接到该区域的怪物列表
	areaToMonsters map[int][]int
	// 缓存结果: defeatedMonsters位掩码 -> 可达区域map
	cache map[int64]map[int]bool
	// 缓存LRU，防止内存无限增长
	cacheOrder   []int64
	maxCacheSize int
}

// 初始化缓存
func NewAccessibilityCache(allMonsters []*GlobalMonster, maxCacheSize int) *AccessibilityCache {
	cache := &AccessibilityCache{
		monsterToAreas: make(map[int][]int),
		areaToMonsters: make(map[int][]int),
		cache:          make(map[int64]map[int]bool),
		cacheOrder:     make([]int64, 0),
		maxCacheSize:   maxCacheSize,
	}

	// 预计算怪物-区域映射关系
	for monsterIdx, monster := range allMonsters {
		cache.monsterToAreas[monsterIdx] = monster.ConnectedAreas

		for _, areaID := range monster.ConnectedAreas {
			cache.areaToMonsters[areaID] = append(cache.areaToMonsters[areaID], monsterIdx)
		}
	}

	return cache
}

// 清理LRU缓存
func (ac *AccessibilityCache) evictOldEntries() {
	if len(ac.cache) <= ac.maxCacheSize {
		return
	}

	// 删除最旧的条目
	toRemove := len(ac.cache) - ac.maxCacheSize + 1
	for i := 0; i < toRemove && len(ac.cacheOrder) > 0; i++ {
		oldKey := ac.cacheOrder[0]
		delete(ac.cache, oldKey)
		ac.cacheOrder = ac.cacheOrder[1:]
	}
}

// 获取可达区域（带缓存）
func (ac *AccessibilityCache) GetAccessibleAreas(defeatedMonsters int64, startArea int) map[int]bool {
	// 检查缓存
	if cached, exists := ac.cache[defeatedMonsters]; exists {
		// 移动到LRU队列末尾
		for i, key := range ac.cacheOrder {
			if key == defeatedMonsters {
				ac.cacheOrder = append(ac.cacheOrder[:i], ac.cacheOrder[i+1:]...)
				break
			}
		}
		ac.cacheOrder = append(ac.cacheOrder, defeatedMonsters)
		return cached
	}

	// 计算可达区域
	accessible := ac.calculateAccessibleAreas(defeatedMonsters, startArea)

	// 存入缓存
	ac.cache[defeatedMonsters] = accessible
	ac.cacheOrder = append(ac.cacheOrder, defeatedMonsters)
	ac.evictOldEntries()

	return accessible
}

// 实际计算可达区域（优化版）
func (ac *AccessibilityCache) calculateAccessibleAreas(defeatedMonsters int64, startArea int) map[int]bool {
	accessible := make(map[int]bool)
	accessible[startArea] = true
	queue := []int{startArea}

	for len(queue) > 0 {
		currentArea := queue[0]
		queue = queue[1:]

		// 检查从当前区域能通过哪些已击败的怪物到达新区域
		if monsters, exists := ac.areaToMonsters[currentArea]; exists {
			for _, monsterIdx := range monsters {
				if hasBit(defeatedMonsters, monsterIdx) { // 怪物已被击败
					// 该怪物连接的所有区域都变为可访问
					for _, connectedAreaID := range ac.monsterToAreas[monsterIdx] {
						if !accessible[connectedAreaID] {
							accessible[connectedAreaID] = true
							queue = append(queue, connectedAreaID)
						}
					}
				}
			}
		}
	}

	return accessible
}

// 增量更新可达区域（当击败新怪物时）
func (ac *AccessibilityCache) GetAccessibleAreasIncremental(
	baseDefeatedMonsters int64,
	newlyDefeatedMonster int,
	startArea int,
	baseAccessible map[int]bool) map[int]bool {

	newDefeatedMonsters := setBit(baseDefeatedMonsters, newlyDefeatedMonster)

	// 检查缓存
	if cached, exists := ac.cache[newDefeatedMonsters]; exists {
		return cached
	}

	// 增量计算：基于已有的可达区域，只处理新击败怪物的影响
	newAccessible := make(map[int]bool)
	for area, reachable := range baseAccessible {
		newAccessible[area] = reachable
	}

	// 检查新击败的怪物能带来哪些新的可达区域
	queue := []int{}

	// 检查新击败怪物连接的区域
	if areas, exists := ac.monsterToAreas[newlyDefeatedMonster]; exists {
		for _, areaID := range areas {
			if newAccessible[areaID] {
				// 如果怪物连接的区域已经可达，则怪物连接的所有区域都变为可达
				for _, connectedAreaID := range areas {
					if !newAccessible[connectedAreaID] {
						newAccessible[connectedAreaID] = true
						queue = append(queue, connectedAreaID)
					}
				}
				break
			}
		}
	}

	// 从新可达的区域继续扩展
	for len(queue) > 0 {
		currentArea := queue[0]
		queue = queue[1:]

		if monsters, exists := ac.areaToMonsters[currentArea]; exists {
			for _, monsterIdx := range monsters {
				if hasBit(newDefeatedMonsters, monsterIdx) {
					for _, connectedAreaID := range ac.monsterToAreas[monsterIdx] {
						if !newAccessible[connectedAreaID] {
							newAccessible[connectedAreaID] = true
							queue = append(queue, connectedAreaID)
						}
					}
				}
			}
		}
	}

	// 存入缓存
	ac.cache[newDefeatedMonsters] = newAccessible
	ac.cacheOrder = append(ac.cacheOrder, newDefeatedMonsters)
	ac.evictOldEntries()

	return newAccessible
}

// 优化后的主函数
func findOptimalPath(graph *Graph, startArea, endArea int, initialHP int16, initialATK, initialDEF, initialYellowKeys, requiredATK, requiredDEF, requiredYellowKeys int8) SearchResult {
	// 获取所有怪物和宝物
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

	// 应用宝物效果（无变化）
	applyTreasures := func(hp int16, atk, def, yellowKey int8, treasureIndices []int) (int16, int8, int8, int8) {
		newHP, newATK, newDEF, newYellowKeys := hp, atk, def, yellowKey
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
			}
		}
		return newHP, newATK, newDEF, newYellowKeys
	}

	// 状态编码
	encodeState := func(defeatedMonsters int64, yellowKeys int8) int64 {
		if yellowKeys > 15 {
			yellowKeys = 15
		}
		return (int64(yellowKeys) << 60) | defeatedMonsters
	}

	// DP表
	dp := make(map[int64]*State)

	// 初始状态
	var initialDefeated int64 = 0
	var initialCollected int64 = 0
	initialAccessible := accessCache.GetAccessibleAreas(initialDefeated, startArea)
	initialCollectible := getCollectibleTreasuresOptimized(initialAccessible, initialCollected)

	newHP, newATK, newDEF, newYellowKeys := applyTreasures(initialHP, initialATK, initialDEF, initialYellowKeys, initialCollectible)
	newInitialCollected := initialCollected
	for _, idx := range initialCollectible {
		newInitialCollected = setBit(newInitialCollected, idx)
	}

	initialStateKey := encodeState(initialDefeated, newYellowKeys)
	path := []int16{}

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
	queue := []int64{initialStateKey}
	visited := make(map[int64]bool)
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

		// 使用缓存获取可达区域
		accessibleAreas := accessCache.GetAccessibleAreas(state.DefeatedMonsters, startArea)

		// 检查胜利条件
		if accessibleAreas[endArea] && state.ATK >= requiredATK && state.DEF >= requiredDEF && state.YellowKeys >= requiredYellowKeys {
			if bestResult == nil || state.HP > bestResult.HP {
				newPath := make([]int16, len(state.Path))
				copy(newPath, state.Path)

				bestResult = &SearchResult{
					HP:             state.HP,
					ATK:            state.ATK,
					DEF:            state.DEF,
					YellowKeys:     state.YellowKeys,
					Path:           newPath,
					DefeatedCount:  countBits(state.DefeatedMonsters),
					CollectedCount: countBits(state.CollectedTreasures),
				}
			}
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

			if monster.ID == 81 && state.YellowKeys <= 0 {
				continue
			}

			damage := getDamage(state.ATK, state.DEF, monster.ID)
			if damage >= state.HP {
				continue
			}

			newHP := state.HP - damage
			newYellowKeys := state.YellowKeys
			newDefeated := setBit(state.DefeatedMonsters, monsterIdx)

			if monster.ID == 81 && state.YellowKeys >= 0 {
				newYellowKeys -= 1
			}

			// 使用增量更新获取新的可达区域
			newAccessible := accessCache.GetAccessibleAreasIncremental(
				state.DefeatedMonsters, monsterIdx, startArea, accessibleAreas)

			newCollectible := getCollectibleTreasuresOptimized(newAccessible, state.CollectedTreasures)

			finalHP, finalATK, finalDEF, finalYK := applyTreasures(newHP, state.ATK, state.DEF, newYellowKeys, newCollectible)
			finalCollected := state.CollectedTreasures
			for _, idx := range newCollectible {
				finalCollected = setBit(finalCollected, idx)
			}

			newStateKey := encodeState(newDefeated, finalYK)

			existingState, exists := dp[newStateKey]
			if !exists || finalHP > existingState.HP {
				newPath := make([]int16, len(state.Path))
				copy(newPath, state.Path)
				newPath = append(newPath, int16(damage))

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
		fmt.Printf("缓存命中统计: 总计算次数约 %d\n", iterations)
		printPath(bestResult.Path)
		return *bestResult
	} else {
		return SearchResult{
			HP:      -1,
			Path:    []int16{},
			Message: fmt.Sprintf("无法达到要求的属性 (ATK>=%d, DEF>=%d) 并到达终点", requiredATK, requiredDEF),
		}
	}
}
