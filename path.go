package main

import (
	"fmt"
	"sort"
)

type State struct {
	HP                 int16    // 2字节
	ATK                int8     // 1字节
	DEF                int8     // 1字节
	YellowKeys         int8     // 1字节
	BlueKeys           int8     // 1字节 - 新增蓝钥匙
	DefeatedMonsters   int64    // 8字节
	CollectedTreasures int64    // 8字节
	PrevKey            int64    // 新增：前驱状态的key
	Action             [2]int16 // 新增：当前动作 [damage, pos编码]
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

// 怪物ID常量
const (
	YellowDoorID = 81 // 黄门
	BlueDoorID   = 82 // 蓝门
)

type PriorityQueue []*StateNode

type StateNode struct {
	Key   int64
	HP    int16
	Index int
}

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].HP > pq[j].HP } // 最大堆
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i]; pq[i].Index, pq[j].Index = i, j }

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*StateNode)
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

// reconstructPath: 回溯生成完整路径
func reconstructPath(dp map[int64]*State, endKey int64) []int16 {
	path := []int16{}
	for key := endKey; key != 0; {
		state := dp[key]
		if state == nil || (state.Action[0] == 0 && state.Action[1] == 0 && state.PrevKey == 0) {
			break
		}
		path = append([]int16{state.Action[0], state.Action[1]}, path...)
		key = state.PrevKey
	}
	return path
}

// 展示状态编码示例（调试用）
func showStateEncoding(defeatedMonsters int64, yellowKeys, blueKeys int8) {
	encoded := (int64(blueKeys) << 62) | (int64(yellowKeys) << 59) | defeatedMonsters
	fmt.Printf("怪物状态: %059b\n", defeatedMonsters)
	fmt.Printf("黄钥匙数: %d (二进制: %03b)\n", yellowKeys, yellowKeys)
	fmt.Printf("蓝钥匙数: %d (二进制: %02b)\n", blueKeys, blueKeys)
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
					// 该怪物连接的所有区域和怪物自身区域都变为可访问
					// 添加怪物自身区域作为可达区域
					monsterAreaID := monsterIdx + len(ac.areaToMonsters)
					accessible[monsterAreaID] = true
					queue = append(queue, monsterAreaID)
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
				// 如果怪物连接的区域已经可达，则怪物连接的所有区域和怪物自身区域都变为可达
				// 添加怪物自身区域作为可达区域
				monsterAreaID := newlyDefeatedMonster + len(ac.areaToMonsters)
				newAccessible[monsterAreaID] = true
				queue = append(queue, monsterAreaID)
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
					// 添加怪物自身区域作为可达区域
					monsterAreaID := monsterIdx + len(ac.areaToMonsters)
					newAccessible[monsterAreaID] = true
					queue = append(queue, monsterAreaID)
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

type HeroItem struct {
	AreaID     int
	HP         int16
	ATK        int8
	DEF        int8
	YellowKeys int8
	BlueKeys   int8
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
		if yellowKeys > 7 {
			yellowKeys = 7
			_ = fmt.Errorf("黄钥匙数量超过最大限制，已调整为7")
		}
		if blueKeys > 3 {
			blueKeys = 3
			_ = fmt.Errorf("蓝钥匙数量超过最大限制，已调整为3")
		}
		return (int64(blueKeys) << 62) | (int64(yellowKeys) << 59) | defeatedMonsters
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
	}

	// BFS搜索（分阶段：第一阶段只扩展状态，第二阶段统一选最优解）
	queue := []int64{initialStateKey}
	visited := make(map[int64]bool)
	visited[initialStateKey] = true

	var candidateKeys []int64 // 阶段一收集所有满足终点条件的状态key
	var iterations int64
	var maxIterations int64
	maxIterations = 1 << 50

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
		fmt.Printf("\n=== 找到最优解 ===\n")
		fmt.Printf("最终属性: HP=%d, ATK=%d, DEF=%d, 黄钥匙=%d, 蓝钥匙=%d\n",
			bestResult.HP, bestResult.ATK, bestResult.DEF, bestResult.YellowKeys, bestResult.BlueKeys)
		fmt.Printf("缓存命中统计: 总计算次数约 %d\n", iterations)
		printPath(bestResult.Path)
		return *bestResult
	} else {
		fmt.Printf("\n=== 找不到最优解 ===\n")
		return SearchResult{
			HP:   -1,
			Path: []int16{},
		}
	}
}
