package main

import (
	"fmt"
	"runtime"
	"time"
)

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

func main() {
	startTime := time.Now()
	initDamageCache()

	// 创建转换器并运行测试
	converter := NewMapToGraphConverter(gameMap, treasureMap, monsterMap, start, end)
	length := len(gameMap)
	for i, row := range gameMap {
		fmt.Print("[")
		for j, val := range row {
			fmt.Printf("%3d", val)
			if j < length-1 {
				fmt.Print(",")
			}
		}
		fmt.Print("]")
		if i < length-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	graph := converter.Convert()
	for _, point := range graph.BreakPoints {
		fmt.Printf("BreakPoint at %v, AreaIDs: %v\n", point.Pos, point.AreaIDs)
	}
	maxHP := int16(0)
	var maxResult SearchResult
	var bestPoint [2]int
	for _, point := range graph.BreakPoints {
		x := point.Pos[0]
		y := point.Pos[1]
		gameMap[x][y] = 0
		newConvertor := NewMapToGraphConverter(gameMap, treasureMap, monsterMap, start, end)
		newGraph := newConvertor.Convert()
		result := findOptimalPath(newGraph, &HeroItem{
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
		if result.HP > maxHP {
			maxHP = result.HP
			maxResult = result
			bestPoint = point.Pos
		}
		gameMap[x][y] = 1
	}
	if maxHP > 0 {
		fmt.Printf("\n=== 找到最优解 ===\n")
		fmt.Printf("最终属性: HP=%d", maxResult.HP)
		fmt.Printf("破点：%v", bestPoint)
		printPath(maxResult.Path)
	} else {
		fmt.Printf("\n=== 找不到最优解 ===\n")
	}
	//graph.Print()

	endTime := time.Now()
	fmt.Printf("Execution time: %v seconds\n", endTime.Sub(startTime).Seconds())
	printStats()
}
