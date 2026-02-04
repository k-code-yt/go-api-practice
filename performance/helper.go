package main

import (
	"fmt"
	"runtime"
)

func PrintMemStats(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\n========= %s =========\n", label)

	fmt.Printf("Alloc       = %s (currently in use)\n", bToMb(m.Alloc))
	fmt.Printf("TotalAlloc  = %s (cumulative allocated)\n", bToMb(m.TotalAlloc))

	fmt.Printf("HeapObjects = %v (live objects)\n", m.HeapObjects)
	fmt.Printf("NumGC       = %v (GC cycles)\n", m.NumGC)
	fmt.Printf("GCCPUFrac   = %.4f%% (CPU time in GC)\n", m.GCCPUFraction*100)

	fmt.Printf("\nMemory breakdown:\n")
	fmt.Printf("  HeapInuse   = %s\n", bToMb(m.HeapInuse))
	fmt.Printf("  HeapIdle    = %s\n", bToMb(m.HeapIdle))
	fmt.Printf("  StackInuse  = %s\n", bToMb(m.StackInuse))

	liveObjects := m.Mallocs - m.Frees
	fmt.Printf("\nAllocation stats:\n")
	fmt.Printf("  Mallocs     = %v (total allocations)\n", m.Mallocs)
	fmt.Printf("  Frees       = %v (total frees)\n", m.Frees)
	fmt.Printf("  Live        = %v (current objects)\n", liveObjects)
}

func bToMb(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
