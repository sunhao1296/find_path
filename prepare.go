package main

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
