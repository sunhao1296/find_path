package main

import (
	"fmt"
	"runtime"
	"time"
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
	startTime := time.Now()
	initDamageCache()

	// 创建转换器并运行测试
	converter := NewMapToGraphConverter(gameMap, treasureMap, monsterMap, start, end)
	graph := converter.Convert()
	findOptimalPath(graph, graph.StartArea, graph.EndArea,
		240, 8, 8, 2, 15, 15, 0)
	endTime := time.Now()
	fmt.Printf("Execution time: %v seconds\n", endTime.Sub(startTime).Seconds())
	printStats()
}
