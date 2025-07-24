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
	findOptimalPath(graph, &HeroItem{
		HP:         280,
		ATK:        8,
		DEF:        8,
		AreaID:     graph.StartArea,
		YellowKeys: 0,
		BlueKeys:   0,
	}, &HeroItem{
		ATK:        15,
		DEF:        15,
		AreaID:     graph.EndArea,
		YellowKeys: 0,
		BlueKeys:   0,
	})
	endTime := time.Now()
	fmt.Printf("Execution time: %v seconds\n", endTime.Sub(startTime).Seconds())
	printStats()
}
