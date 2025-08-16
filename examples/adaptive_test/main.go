package main

import (
	"fmt"
	"time"

	"github.com/shaban/macaudio"
)

func main() {
	fmt.Println("MacAudio Adaptive Polling Test")
	fmt.Println("==============================")
	
	// Create minimal engine
	config := macaudio.EngineConfig{
		BufferSize:   256,
		SampleRate:   44100.0,
		ErrorHandler: &macaudio.DefaultErrorHandler{},
	}
	
	engine, err := macaudio.NewEngine(config)
	if err != nil {
		panic(err)
	}
	
	if err := engine.Start(); err != nil {
		panic(err)
	}
	defer engine.Stop()
	
	monitor := engine.GetDeviceMonitor()
	
	fmt.Printf("Initial polling interval: %v\n", monitor.GetPollingInterval())
	fmt.Println("Monitoring device polling behavior for 10 seconds...")
	fmt.Println("(No device changes expected - watch interval adapt)")
	
	// Monitor performance every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	start := time.Now()
	for i := 0; i < 5; i++ {
		<-ticker.C
		
		avgTime, maxTime, checkCount := monitor.GetPerformanceStats()
		interval := monitor.GetPollingInterval()
		
		fmt.Printf("[%2.0fs] Interval: %5s | Avg: %6s | Max: %6s | Checks: %4d\n",
			time.Since(start).Seconds(),
			interval.String(),
			avgTime.String(),
			maxTime.String(),
			checkCount,
		)
	}
	
	// Setup graceful shutdown
	fmt.Println("\nTest complete. Final stats:")
	avgTime, maxTime, checkCount := monitor.GetPerformanceStats()
	interval := monitor.GetPollingInterval()
	
	fmt.Printf("Final Interval: %s\n", interval)
	fmt.Printf("Average Check Time: %s\n", avgTime)
	fmt.Printf("Maximum Check Time: %s\n", maxTime)
	fmt.Printf("Total Checks: %d\n", checkCount)
	
	if avgTime > 0 {
		efficiency := (50 * time.Microsecond * 100) / avgTime
		if efficiency > 100 {
			efficiency = 100
		}
		fmt.Printf("Target Efficiency: %d%% (target: 50μs, actual: %s)\n", efficiency, avgTime)
	}
	
	cpuUsage := float64(avgTime) / float64(interval) * 100
	fmt.Printf("CPU Usage: %.3f%% (check_time/interval)\n", cpuUsage)
	
	checksPerSecond := float64(checkCount) / 10.0
	fmt.Printf("Checks per second: %.1f\n", checksPerSecond)
}
