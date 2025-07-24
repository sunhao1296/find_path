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
