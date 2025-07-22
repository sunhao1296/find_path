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

// 转换地图为图
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

	// 第一遍遍历：标记连通区域
	for i := 0; i < c.rows; i++ {
		for j := 0; j < c.cols; j++ {
			cellValue := c.gameMap[i][j]

			// 跳过墙、已访问位置和怪物位置
			if visited[i][j] != -1 || cellValue == 1 {
				continue
			}
			if _, exists := c.monsterMap[cellValue]; exists {
				continue
			}

			areaID := areaCount
			queue := [][2]int{{i, j}}
			visited[i][j] = areaID
			area := &Area{
				ID:        areaID,
				Treasures: []*TreasureItem{},
				Neighbors: []*Neighbor{},
				Positions: [][2]int{},
			}

			for len(queue) > 0 {
				pos := queue[0]
				queue = queue[1:]
				x, y := pos[0], pos[1]
				area.Positions = append(area.Positions, [2]int{x, y})

				// 检查是否为起点或终点
				if x == c.start[0] && y == c.start[1] {
					startArea = areaID
				}
				if x == c.end[0] && y == c.end[1] {
					endArea = areaID
				}

				// 检查是否有宝物
				cellVal := c.gameMap[x][y]
				if treasure, exists := c.treasureMap[cellVal]; exists {
					area.Treasures = append(area.Treasures, &TreasureItem{
						ID:         len(area.Treasures),
						Type:       treasure.Type,
						Value:      treasure.Value,
						OriginalID: cellVal,
					})
				}

				// 探索四个方向
				for _, dir := range c.directions {
					nx, ny := x+dir[0], y+dir[1]
					if nx < 0 || nx >= c.rows || ny < 0 || ny >= c.cols {
						continue
					}
					if visited[nx][ny] != -1 || c.gameMap[nx][ny] == 1 {
						continue
					}

					neighborVal := c.gameMap[nx][ny]
					if _, exists := c.monsterMap[neighborVal]; exists {
						// 记录怪物连接
						key := fmt.Sprintf("%d,%d", nx, ny)
						if c.monsterConnections[key] == nil {
							c.monsterConnections[key] = make(map[int]bool)
						}
						c.monsterConnections[key][areaID] = true
					} else {
						// 空地或宝物，加入当前区域
						visited[nx][ny] = areaID
						queue = append(queue, [2]int{nx, ny})
					}
				}
			}
			c.areas = append(c.areas, area)
			areaCount++
		}
	}

	// 第二遍遍历：构建怪物连接信息，避免多重边
	monsterConnections := make(map[string]*MonsterConnection)

	for monsterPos, areas := range c.monsterConnections {
		areaList := make([]int, 0, len(areas))
		for area := range areas {
			areaList = append(areaList, area)
		}

		if len(areaList) >= 2 {
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

			// 为每个相关区域添加到其他所有区域的连接引用
			// 但不创建实际的重复边，而是引用同一个怪物
			for _, areaID := range areaList {
				// 为每个区域添加一个代表性的邻居，包含所有连接信息
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

	return &Graph{
		Areas:              c.areas,
		StartArea:          startArea,
		EndArea:            endArea,
		AreaMap:            visited,
		MonsterConnections: monsterConnections,
	}
}
