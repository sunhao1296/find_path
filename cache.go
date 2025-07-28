package main

import "fmt"

// 扩展的可达性缓存（需要适配ExtendedBitSet）
type AccessibilityCacheExt struct {
	cache    map[string]map[int]bool
	monsters []*GlobalMonster
	maxSize  int
}

func NewAccessibilityCacheExt(monsters []*GlobalMonster, cacheSize int) *AccessibilityCacheExt {
	return &AccessibilityCacheExt{
		cache:    make(map[string]map[int]bool),
		monsters: monsters,
		maxSize:  cacheSize,
	}
}

func (cache *AccessibilityCacheExt) GetAccessibleAreas(defeated *ExtendedBitSet, startArea int) map[int]bool {
	// 使用defeated的哈希值作为缓存key
	key := fmt.Sprintf("%x", defeated.Hash())

	if result, exists := cache.cache[key]; exists {
		return result
	}

	// 计算可达区域的逻辑（与原来相同，只是用ExtendedBitSet）
	accessible := make(map[int]bool)
	accessible[startArea] = true

	// 遍历所有已击败的怪物，更新可达区域
	for i := 0; i < len(cache.monsters); i++ {
		if defeated.IsSet(i) {
			monster := cache.monsters[i]
			for _, areaID := range monster.ConnectedAreas {
				accessible[areaID] = true
			}
		}
	}

	// 缓存结果
	if len(cache.cache) < cache.maxSize {
		cache.cache[key] = accessible
	}

	return accessible
}

func (cache *AccessibilityCacheExt) GetAccessibleAreasIncremental(oldDefeated *ExtendedBitSet, newMonsterIdx int, startArea int, oldAccessible map[int]bool) map[int]bool {
	// 增量更新：基于旧的可达区域，只添加新击败怪物开启的区域
	newAccessible := make(map[int]bool)

	// 复制原有的可达区域
	for areaID, accessible := range oldAccessible {
		newAccessible[areaID] = accessible
	}

	// 检查边界条件
	if newMonsterIdx < 0 || newMonsterIdx >= len(cache.monsters) {
		return newAccessible
	}

	// 获取新击败的怪物
	newMonster := cache.monsters[newMonsterIdx]

	// 将新怪物连接的区域添加到可达区域中
	for _, areaID := range newMonster.ConnectedAreas {
		newAccessible[areaID] = true
	}

	// 检查是否有连锁反应：新开启的区域可能让其他已击败的怪物变得可达
	// 这种情况在复杂地图中可能出现，比如A区域的怪物开启B区域，B区域有之前击败的怪物连接到C区域
	changed := true
	maxIterations := 10 // 防止无限循环，通常不需要太多次迭代
	iteration := 0

	for changed && iteration < maxIterations {
		changed = false
		iteration++

		// 遍历所有已击败的怪物，检查是否有新的区域变得可达
		for i := 0; i < len(cache.monsters); i++ {
			if !oldDefeated.IsSet(i) && i != newMonsterIdx {
				continue // 只考虑已击败的怪物
			}

			monster := cache.monsters[i]

			// 检查这个怪物是否在任何可达区域中
			monsterReachable := false
			for _, areaID := range monster.ConnectedAreas {
				if newAccessible[areaID] {
					monsterReachable = true
					break
				}
			}

			// 如果怪物可达，确保其连接的所有区域都被标记为可达
			if monsterReachable {
				for _, areaID := range monster.ConnectedAreas {
					if !newAccessible[areaID] {
						newAccessible[areaID] = true
						changed = true // 有新区域被开启，需要再次检查
					}
				}
			}
		}
	}

	// 可选：缓存结果以供后续使用
	newDefeated := oldDefeated.Copy()
	newDefeated.Set(newMonsterIdx)
	cacheKey := fmt.Sprintf("%x", newDefeated.Hash())

	if len(cache.cache) < cache.maxSize {
		cache.cache[cacheKey] = newAccessible
	}

	return newAccessible
}
