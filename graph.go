package main

import (
	"fmt"
	"sort"
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

type BreakPoint struct {
	Pos     [2]int
	AreaIDs []int // 该破点能连接的区域ID
}

// 中心飞目标信息
type CenterFlyTarget struct {
	TargetPos  [2]int // 中心对称点坐标
	TargetArea int    // 目标点所属区域ID
	IsValid    bool   // 目标点是否为有效空地
}

// 中心飞查询结果
type CenterFlyResult struct {
	FromArea    int                      // 起始区域
	Targets     []*CenterFlyTarget       // 所有可达目标
	TargetAreas map[int]*CenterFlyTarget // 按目标区域ID索引的快速查询
}

// Graph 结构添加中心飞相关字段
type Graph struct {
	Areas              []*Area
	StartArea          int
	EndArea            int
	AreaMap            [][]int
	MonsterConnections map[string]*MonsterConnection
	BreakPoints        []*BreakPoint

	// 新增：中心飞相关
	centerPos      [2]int                   // 地图中心坐标
	centerFlyCache map[int]*CenterFlyResult // 按区域ID缓存的中心飞查询结果
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

// 计算给定位置的中心对称点
func (g *Graph) getCenterSymmetricPos(pos [2]int) [2]int {
	return [2]int{
		2*g.centerPos[0] - pos[0],
		2*g.centerPos[1] - pos[1],
	}
}

// 检查位置是否在地图范围内且为空地
func (g *Graph) isValidLandingPos(pos [2]int, gameMap [][]int) bool {
	x, y := pos[0], pos[1]
	rows, cols := len(gameMap), len(gameMap[0])

	// 检查边界
	if x < 0 || x >= rows || y < 0 || y >= cols {
		return false
	}

	// 检查是否为空地（非墙且非怪物）
	cellValue := gameMap[x][y]
	return cellValue != 1 // 1代表墙
}

// 检查位置是否有效
func (c *MapToGraphConverter) isValidPosition(pos [2]int) bool {
	x, y := pos[0], pos[1]
	return x >= 0 && x < c.rows && y >= 0 && y < c.cols && c.gameMap[x][y] != 1
}

// 处理怪物位置，检查其连通性并处理相邻的未访问区域
func (c *MapToGraphConverter) processMonsterPosition(x, y int, visited [][]int, areaCount *int, startArea, endArea *int) {
	connectedAreas := make(map[int]bool)

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

// 修改MapToGraphConverter的Convert方法，添加中心飞缓存构建
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

	// 收集破墙点
	breakPointMap := make(map[string]*BreakPoint)

	for i := 0; i < c.rows; i++ {
		for j := 0; j < c.cols; j++ {
			if c.gameMap[i][j] != 1 {
				continue // 只考虑墙
			}
			neighborAreas := map[int]bool{}
			for _, dir := range c.directions {
				nx, ny := i+dir[0], j+dir[1]
				if nx < 0 || nx >= c.rows || ny < 0 || ny >= c.cols {
					continue
				}
				aid := visited[nx][ny]
				if aid != -1 {
					neighborAreas[aid] = true
				}
			}
			if len(neighborAreas) >= 2 {
				// 生成区域组合key用于去重
				areaList := []int{}
				for aid := range neighborAreas {
					areaList = append(areaList, aid)
				}
				sort.Ints(areaList)
				key := fmt.Sprintf("%v", areaList)
				if _, exists := breakPointMap[key]; !exists {
					breakPointMap[key] = &BreakPoint{
						Pos:     [2]int{i, j},
						AreaIDs: areaList,
					}
				}
			}
		}
	}
	breakPoints := []*BreakPoint{}
	for _, bp := range breakPointMap {
		breakPoints = append(breakPoints, bp)
	}

	// 验证转换结果
	//c.validateConversion()

	// 创建Graph
	graph := &Graph{
		Areas:              c.areas,
		StartArea:          startArea,
		EndArea:            endArea,
		AreaMap:            visited,
		MonsterConnections: monsterConnections,
		BreakPoints:        breakPoints,
	}

	// 构建中心飞缓存
	graph.buildCenterFlyCache(c.gameMap)
	//ExampleCenterFlyUsage(graph)
	return graph
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
	fmt.Printf("总区域数: %d\n", len(c.areas))
	fmt.Printf("怪物连接数: %d\n", len(c.monsterConnections))

	if totalTreasuresInMap != totalTreasuresInAreas {
		fmt.Printf("警告：有 %d 个宝物丢失！\n", totalTreasuresInMap-totalTreasuresInAreas)
	}
}

// 构建中心飞查询缓存
func (g *Graph) buildCenterFlyCache(gameMap [][]int) {
	g.centerFlyCache = make(map[int]*CenterFlyResult)

	// 计算地图中心
	rows, cols := len(gameMap), len(gameMap[0])
	g.centerPos = [2]int{rows / 2, cols / 2}

	// 为每个区域构建中心飞目标
	for _, area := range g.Areas {
		result := &CenterFlyResult{
			FromArea:    area.ID,
			Targets:     []*CenterFlyTarget{},
			TargetAreas: make(map[int]*CenterFlyTarget),
		}

		// 检查该区域所有位置的中心对称点
		targetAreaSet := make(map[int]bool)

		for _, pos := range area.Positions {
			symmetricPos := g.getCenterSymmetricPos(pos)

			if g.isValidLandingPos(symmetricPos, gameMap) {
				// 查找目标位置属于哪个区域
				targetAreaID := g.AreaMap[symmetricPos[0]][symmetricPos[1]]

				if targetAreaID != -1 && !targetAreaSet[targetAreaID] {
					targetAreaSet[targetAreaID] = true

					target := &CenterFlyTarget{
						TargetPos:  symmetricPos,
						TargetArea: targetAreaID,
						IsValid:    true,
					}

					result.Targets = append(result.Targets, target)
					result.TargetAreas[targetAreaID] = target
				}
			}
		}

		g.centerFlyCache[area.ID] = result
	}
}

// 查询从指定区域出发的所有中心飞目标
func (g *Graph) GetCenterFlyTargets(fromAreaID int) *CenterFlyResult {
	if result, exists := g.centerFlyCache[fromAreaID]; exists {
		return result
	}
	return &CenterFlyResult{
		FromArea:    fromAreaID,
		Targets:     []*CenterFlyTarget{},
		TargetAreas: make(map[int]*CenterFlyTarget),
	}
}

// 检查是否可以从一个区域中心飞到另一个区域
func (g *Graph) CanCenterFlyTo(fromAreaID, toAreaID int) bool {
	result := g.GetCenterFlyTargets(fromAreaID)
	_, canFly := result.TargetAreas[toAreaID]
	return canFly
}

// 获取从指定区域可以中心飞到的所有区域ID列表
func (g *Graph) GetCenterFlyReachableAreas(fromAreaID int) []int {
	result := g.GetCenterFlyTargets(fromAreaID)
	areas := make([]int, 0, len(result.Targets))

	for _, target := range result.Targets {
		areas = append(areas, target.TargetArea)
	}

	return areas
}

// 查找能够飞到指定区域的所有源区域
func (g *Graph) GetCenterFlySourceAreas(toAreaID int) []int {
	sources := []int{}

	for fromAreaID := range g.centerFlyCache {
		if g.CanCenterFlyTo(fromAreaID, toAreaID) {
			sources = append(sources, fromAreaID)
		}
	}

	return sources
}

// 打印中心飞信息（调试用）
func (g *Graph) PrintCenterFlyInfo() {
	fmt.Printf("=== 中心飞查询信息 ===\n")
	fmt.Printf("地图中心: %v\n", g.centerPos)

	for areaID, result := range g.centerFlyCache {
		if len(result.Targets) > 0 {
			fmt.Printf("区域 %d 可中心飞到: ", areaID)
			for i, target := range result.Targets {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("区域%d@%v", target.TargetArea, target.TargetPos)
			}
			fmt.Println()
		}
	}
}
