package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Area struct {
	ID        int
	Treasures []*TreasureItem
	Neighbors []*Neighbor
	Positions [][2]int
}

type Neighbor struct {
	Area       int
	MonsterID  int
	Monster    *Monster
	MonsterPos [2]int
}

type Graph struct {
	Areas              []*Area
	StartArea          int
	EndArea            int
	AreaMap            [][]int
	MonsterConnections map[string]*MonsterConnection // key为"x,y"格式的怪物位置
}

// 地图转图转换器
type MapToGraphConverter struct {
	gameMap            [][]int
	rows               int
	cols               int
	treasureMap        map[int]*Treasure
	monsterMap         map[int]*Monster
	start              [2]int
	end                [2]int
	directions         [][2]int
	areas              []*Area
	monsterConnections map[string]map[int]bool
}

// 创建新的转换器
func NewMapToGraphConverter(gameMap [][]int, treasureMap map[int]*Treasure, monsterMap map[int]*Monster, start, end [2]int) *MapToGraphConverter {
	return &MapToGraphConverter{
		gameMap:            gameMap,
		rows:               len(gameMap),
		cols:               len(gameMap[0]),
		treasureMap:        treasureMap,
		monsterMap:         monsterMap,
		start:              start,
		end:                end,
		directions:         [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}},
		areas:              []*Area{},
		monsterConnections: make(map[string]map[int]bool),
	}
}

// 检查位置是否有效
func (c *MapToGraphConverter) isValidPosition(pos [2]int) bool {
	x, y := pos[0], pos[1]
	return x >= 0 && x < c.rows && y >= 0 && y < c.cols && c.gameMap[x][y] != 1
}

// 处理怪物位置，将怪物本身作为单点区域并检查连通性
func (c *MapToGraphConverter) processMonsterPosition(x, y int, visited [][]int, areaCount *int, startArea, endArea *int) {
	connectedAreas := make(map[int]bool)

	// 创建怪物自身的单点区域
	monsterAreaID := *areaCount
	c.processCellAsNewArea(x, y, monsterAreaID, visited, startArea, endArea)
	connectedAreas[monsterAreaID] = true
	*areaCount++

	// 检查怪物四周的连通性
	for _, dir := range c.directions {
		nx, ny := x+dir[0], y+dir[1]
		if !c.isValidPosition([2]int{nx, ny}) {
			continue
		}

		if visited[nx][ny] != -1 {
			// 邻居已经属于某个区域
			connectedAreas[visited[nx][ny]] = true
		} else {
			// 邻居是未访问的位置
			neighborVal := c.gameMap[nx][ny]
			if _, exists := c.monsterMap[neighborVal]; !exists {
				// 邻居不是怪物，为它创建新区域
				areaID := *areaCount
				c.processCellAsNewArea(nx, ny, areaID, visited, startArea, endArea)
				connectedAreas[areaID] = true
				*areaCount++
			}
		}
	}

	// 记录怪物连接信息
	if len(connectedAreas) >= 1 {
		key := fmt.Sprintf("%d,%d", x, y)
		c.monsterConnections[key] = connectedAreas
	}
}

// 将指定位置作为新区域进行BFS扩展
func (c *MapToGraphConverter) processCellAsNewArea(x, y, areaID int, visited [][]int, startArea, endArea *int) {
	queue := [][2]int{{x, y}}
	visited[x][y] = areaID
	area := &Area{
		ID:        areaID,
		Treasures: []*TreasureItem{},
		Neighbors: []*Neighbor{},
		Positions: [][2]int{},
	}

	for len(queue) > 0 {
		pos := queue[0]
		queue = queue[1:]
		px, py := pos[0], pos[1]
		area.Positions = append(area.Positions, [2]int{px, py})

		// 检查是否为起点或终点
		if startArea != nil && px == c.start[0] && py == c.start[1] {
			*startArea = areaID
		}
		if endArea != nil && px == c.end[0] && py == c.end[1] {
			*endArea = areaID
		}

		// 检查是否有宝物
		cellVal := c.gameMap[px][py]
		if treasure, exists := c.treasureMap[cellVal]; exists {
			area.Treasures = append(area.Treasures, &TreasureItem{
				ID:         len(area.Treasures),
				Type:       treasure.Type,
				Value:      treasure.Value,
				OriginalID: cellVal,
			})
		}

		// BFS扩展（只扩展到非怪物位置）
		for _, dir := range c.directions {
			nx, ny := px+dir[0], py+dir[1]
			if !c.isValidPosition([2]int{nx, ny}) || visited[nx][ny] != -1 {
				continue
			}

			neighborVal := c.gameMap[nx][ny]
			if _, exists := c.monsterMap[neighborVal]; !exists {
				// 非怪物位置，加入当前区域
				visited[nx][ny] = areaID
				queue = append(queue, [2]int{nx, ny})
			}
			// 怪物位置不加入queue，但会在后续的processMonsterPosition中处理
		}
	}

	c.areas = append(c.areas, area)
}

// 构建怪物连接信息
func (c *MapToGraphConverter) buildMonsterConnections(visited [][]int) map[string]*MonsterConnection {
	monsterConnections := make(map[string]*MonsterConnection)

	for monsterPos, areas := range c.monsterConnections {
		areaList := make([]int, 0, len(areas))
		for area := range areas {
			areaList = append(areaList, area)
		}

		if len(areaList) >= 1 { // 修改：至少连接1个区域就有意义
			parts := strings.Split(monsterPos, ",")
			x, _ := strconv.Atoi(parts[0])
			y, _ := strconv.Atoi(parts[1])
			monsterID := c.gameMap[x][y]
			monster := c.monsterMap[monsterID]

			// 创建怪物连接信息
			monsterConnections[monsterPos] = &MonsterConnection{
				MonsterID:      monsterID,
				Monster:        monster,
				MonsterPos:     [2]int{x, y},
				ConnectedAreas: areaList,
			}

			// 为每个相关区域添加邻居引用
			for _, areaID := range areaList {
				neighbor := &Neighbor{
					Area:       -1, // 特殊标记，表示这是一个多区域连接
					MonsterID:  monsterID,
					Monster:    monster,
					MonsterPos: [2]int{x, y},
				}
				c.areas[areaID].Neighbors = append(c.areas[areaID].Neighbors, neighbor)
			}
		}
	}

	return monsterConnections
}

// 验证转换结果
func (c *MapToGraphConverter) validateConversion() {
	totalTreasuresInMap := 0
	for i := 0; i < c.rows; i++ {
		for j := 0; j < c.cols; j++ {
			if _, exists := c.treasureMap[c.gameMap[i][j]]; exists {
				totalTreasuresInMap++
			}
		}
	}

	totalTreasuresInAreas := 0
	for _, area := range c.areas {
		totalTreasuresInAreas += len(area.Treasures)
	}

	fmt.Printf("地图中宝物总数: %d\n", totalTreasuresInMap)
	fmt.Printf("区域中宝物总数: %d\n", totalTreasuresInAreas)
	fmt.Printf("总区域数: %d\n", len(c.areas))
	fmt.Printf("怪物连接数: %d\n", len(c.monsterConnections))

	if totalTreasuresInMap != totalTreasuresInAreas {
		fmt.Printf("警告：有 %d 个宝物丢失！\n", totalTreasuresInMap-totalTreasuresInAreas)

		// 打印丢失宝物的位置
		for i := 0; i < c.rows; i++ {
			for j := 0; j < c.cols; j++ {
				if _, exists := c.treasureMap[c.gameMap[i][j]]; exists {
					found := false
					for _, area := range c.areas {
						for _, pos := range area.Positions {
							if pos[0] == i && pos[1] == j {
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					if !found {
						fmt.Printf("丢失的宝物位置: (%d,%d), 值=%d\n", i, j, c.gameMap[i][j])
					}
				}
			}
		}
	} else {
		fmt.Println("✓ 所有宝物都已正确分配到区域")
	}
}

// 转换地图为图 - 修复版
func (c *MapToGraphConverter) Convert() *Graph {
	visited := make([][]int, c.rows)
	for i := range visited {
		visited[i] = make([]int, c.cols)
		for j := range visited[i] {
			visited[i][j] = -1
		}
	}

	areaCount := 0
	startArea := -1
	endArea := -1

	// 第一遍遍历：处理所有位置
	for i := 0; i < c.rows; i++ {
		for j := 0; j < c.cols; j++ {
			cellValue := c.gameMap[i][j]

			// 跳过墙和已访问位置
			if visited[i][j] != -1 || cellValue == 1 {
				continue
			}

			// 关键修改：特殊处理怪物位置
			if _, exists := c.monsterMap[cellValue]; exists {
				// 怪物位置：检查连通性并处理相邻区域
				c.processMonsterPosition(i, j, visited, &areaCount, &startArea, &endArea)
				continue
			}

			// 处理普通区域（空地、宝物等）
			areaID := areaCount
			c.processCellAsNewArea(i, j, areaID, visited, &startArea, &endArea)
			areaCount++
		}
	}

	// 第二遍：构建最终的怪物连接信息
	monsterConnections := c.buildMonsterConnections(visited)

	// 验证转换结果
	c.validateConversion()

	return &Graph{
		Areas:              c.areas,
		StartArea:          startArea,
		EndArea:            endArea,
		AreaMap:            visited,
		MonsterConnections: monsterConnections,
	}
}