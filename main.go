package main

import (
	"fmt"
	"runtime"
)

func printStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Printf("Alloc = %v KB\n", memStats.Alloc/1024)
	fmt.Printf("TotalAlloc = %v KB\n", memStats.TotalAlloc/1024)
	fmt.Printf("Sys = %v KB\n", memStats.Sys/1024)
	fmt.Printf("NumGC = %v\n", memStats.NumGC)
	numCPU := runtime.NumCPU()
	fmt.Printf("NumCPU = %d\n", numCPU)
}

func main() {
	initDamageCache()

	// 创建转换器并运行测试
	converter := NewMapToGraphConverter(gameMap, treasureMap, monsterMap, start, end)
	graph := converter.Convert()
	result := findOptimalPath(graph, graph.StartArea, graph.EndArea,
		240, 8, 8, 0, 15, 15, 0)

	fmt.Printf("最终属性: HP=%d\n", result.HP)
	fmt.Printf("\n路径步骤:\n")
	for i, step := range result.Path {
		fmt.Printf("%d. %s\n", i+1, step)
	}
	printStats()
}
