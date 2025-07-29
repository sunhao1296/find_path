package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type HeroItem struct {
	AreaID     int
	HP         int16
	ATK        int8
	DEF        int8
	MDEF       uint8
	Money      uint8
	YellowKeys int8
	BlueKeys   int8
}

type SearchResult struct {
	HP             int16
	Money          uint8
	ATK            int8
	DEF            int8
	MDEF           uint8
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
		if damage == -1 && pos == -1 {
			fmt.Printf("%d. 购买攻击力+1 (花费40金币)\n", i/2+1)
		} else if damage == -2 && pos == -2 {
			fmt.Printf("%d. 购买防御力+1 (花费40金币)\n", i/2+1)
		} else {
			fmt.Printf("%d. 战斗损失%d血, 战斗at %d, %d\n", i/2+1, damage, pos>>8, pos%(1<<8))
		}
	}
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

func printStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	// 打印内存统计信息
	fmt.Printf("Memory Statistics:\n")
	fmt.Printf("HeapAlloc = %v KB\n", memStats.HeapAlloc/1024)
	fmt.Printf("HeapSys = %v KB\n", memStats.HeapSys/1024)
	fmt.Printf("HeapIdle = %v KB\n", memStats.HeapIdle/1024)
	fmt.Printf("HeapInuse = %v KB\n", memStats.HeapInuse/1024)
	fmt.Printf("HeapReleased = %v KB\n", memStats.HeapReleased/1024)
	fmt.Printf("HeapObjects = %v\n", memStats.HeapObjects)
	fmt.Printf("StackInuse = %v KB\n", memStats.StackInuse/1024)
	fmt.Printf("MSpanInuse = %v KB\n", memStats.MSpanInuse/1024)
	fmt.Printf("MCacheInuse = %v KB\n", memStats.MCacheInuse/1024)
	fmt.Printf("BuckHashSys = %v KB\n", memStats.BuckHashSys/1024)
	fmt.Printf("GCSys = %v KB\n", memStats.GCSys/1024)
	fmt.Printf("OtherSys = %v KB\n", memStats.OtherSys/1024)
	fmt.Printf("NextGC = %v KB\n", memStats.NextGC/1024)
	fmt.Printf("PauseTotalNs = %v ns\n", memStats.PauseTotalNs)
	fmt.Printf("Alloc = %v KB\n", memStats.Alloc/1024)
	fmt.Printf("TotalAlloc = %v KB\n", memStats.TotalAlloc/1024)
	fmt.Printf("Sys = %v KB\n", memStats.Sys/1024)
	fmt.Printf("NumGC = %v\n", memStats.NumGC)
	numCPU := runtime.NumCPU()
	fmt.Printf("NumCPU = %d\n", numCPU)
	fmt.Printf("Goroutines = %d\n", runtime.NumGoroutine())
	fmt.Printf("Go version = %s\n", runtime.Version())

}

// 定义任务结构
type task struct {
	point [2]int
}

// 定义结果结构
type result struct {
	hp         int16
	heroResult SearchResult
	point      [2]int
}

func main() {
	startTime := time.Now()
	initDamageCache()

	// 创建转换器并运行测试
	converter := NewMapToGraphConverter(gameMap, treasureMap, monsterMap, start, end)
	length := len(gameMap[0])
	rows := len(gameMap)
	for i, row := range gameMap {
		fmt.Print("[")
		for j, val := range row {
			fmt.Printf("%3d", val)
			if j < length-1 {
				fmt.Print(",")
			}
		}
		fmt.Print("]")
		if i < rows-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	graph := converter.Convert()
	for _, point := range graph.BreakPoints {
		fmt.Printf("BreakPoint at %v, AreaIDs: %v\n", point.Pos, point.AreaIDs)
	}

	// 并发处理逻辑
	var (
		maxHP     = int16(0)
		maxResult SearchResult
		bestPoint [2]int
		mu        sync.Mutex // 用于保护共享变量
	)

	poolSize := 6
	taskCh := make(chan task, len(graph.BreakPoints))
	resultCh := make(chan result, len(graph.BreakPoints))
	var wg sync.WaitGroup

	// 启动 worker pool
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				x, y := t.point[0], t.point[1]
				fmt.Printf("Task at point %v, x: %v, y: %v\n", t.point[0], x, y)
				// 深拷贝 gameMap
				newGameMap := make([][]int, len(gameMap))
				for i := range gameMap {
					newGameMap[i] = make([]int, len(gameMap[i]))
					copy(newGameMap[i], gameMap[i])
				}
				newGameMap[x][y] = 0 // 修改副本

				// 执行原逻辑
				newConvertor := NewMapToGraphConverter(
					newGameMap, treasureMap, monsterMap, start, end,
				)
				newGraph := newConvertor.Convert()
				res := findOptimalPath(newGraph, &HeroItem{
					HP:         initialHP,
					ATK:        initialAtk,
					DEF:        initialDef,
					AreaID:     graph.StartArea,
					YellowKeys: initialYK,
					BlueKeys:   initialBK,
				}, &HeroItem{
					ATK:        requiredATK,
					DEF:        requiredDEF,
					AreaID:     graph.EndArea,
					YellowKeys: requiredYellowKeys,
					BlueKeys:   requiredBlueKeys,
				})

				// 发送结果
				fmt.Printf("point: %v, hp: %v\n", t.point, res.HP)
				resultCh <- result{
					hp:         res.HP,
					heroResult: res,
					point:      t.point,
				}
			}
		}()
	}

	// 发送任务到队列
	go func() {
		for _, point := range graph.BreakPoints {
			taskCh <- task{point: point.Pos}
		}
		close(taskCh)
	}()

	// 启动结果收集协程
	done := make(chan struct{})
	go func() {
		defer close(done)
		for r := range resultCh {
			mu.Lock()
			if r.hp > maxHP {
				maxHP = r.hp
				maxResult = r.heroResult
				bestPoint = r.point
			}
			mu.Unlock()
		}
	}()

	// 等待所有 worker 完成，然后关闭结果通道
	wg.Wait()
	close(resultCh)

	// 等待结果收集完成
	<-done

	//graph.Print()

	if maxHP > 0 {
		fmt.Printf("\n=== 找到最优解 ===\n")
		fmt.Printf("最终属性: HP=%d, Money=%d", maxResult.HP, maxResult.Money)
		fmt.Printf("破点：%v", bestPoint)
		printPath(maxResult.Path)
	} else {
		fmt.Printf("\n=== 找不到最优解 ===\n")
	}
	endTime := time.Now()
	fmt.Printf("Execution time: %v seconds\n", endTime.Sub(startTime).Seconds())
	printStats()
}
