package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

type MemSnapshot struct {
	Alloc         uint64
	TotalAlloc    uint64
	Sys           uint64
	HeapObjects   uint64
	HeapInuse     uint64
	HeapIdle      uint64
	StackInuse    uint64
	Mallocs       uint64
	Frees         uint64
	NumGC         uint32
	GCCPUFraction float64
	Timestamp     time.Time
}

type MemDelta struct {
	AllocDelta         int64
	TotalAllocDelta    int64
	SysDelta           int64
	HeapObjectsDelta   int64
	HeapInuseDelta     int64
	HeapIdleDelta      int64
	StackInuseDelta    int64
	MallocsDelta       int64
	FreesDelta         int64
	NumGCDelta         int32
	GCCPUFractionDelta float64
	Duration           time.Duration
}

type BenchmarkResult struct {
	Name           string
	Implementation string
	Goroutines     int
	Scenario       string
	Before         MemSnapshot
	After          MemSnapshot
	Delta          MemDelta
}

type ComparisonReport struct {
	Timestamp time.Time
	Name      string
	Results   []BenchmarkResult
}

func captureMemSnapshot() MemSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemSnapshot{
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		HeapObjects:   m.HeapObjects,
		HeapInuse:     m.HeapInuse,
		HeapIdle:      m.HeapIdle,
		StackInuse:    m.StackInuse,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		NumGC:         m.NumGC,
		GCCPUFraction: m.GCCPUFraction,
		Timestamp:     time.Now(),
	}
}

func calculateDelta(before, after MemSnapshot) MemDelta {
	return MemDelta{
		AllocDelta:         int64(after.Alloc) - int64(before.Alloc),
		TotalAllocDelta:    int64(after.TotalAlloc) - int64(before.TotalAlloc),
		SysDelta:           int64(after.Sys) - int64(before.Sys),
		HeapObjectsDelta:   int64(after.HeapObjects) - int64(before.HeapObjects),
		HeapInuseDelta:     int64(after.HeapInuse) - int64(before.HeapInuse),
		HeapIdleDelta:      int64(after.HeapIdle) - int64(before.HeapIdle),
		StackInuseDelta:    int64(after.StackInuse) - int64(before.StackInuse),
		MallocsDelta:       int64(after.Mallocs) - int64(before.Mallocs),
		FreesDelta:         int64(after.Frees) - int64(before.Frees),
		NumGCDelta:         int32(after.NumGC) - int32(before.NumGC),
		GCCPUFractionDelta: after.GCCPUFraction - before.GCCPUFraction,
		Duration:           after.Timestamp.Sub(before.Timestamp),
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < 0 {
		return fmt.Sprintf("-%s", formatBytes(-bytes))
	}
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type BenchConfig struct {
	CommitDur     int64
	UpdateDur     int64
	AppendDur     int64
	UpdateRange   int64
	TestDur       time.Duration
	NumGoroutines int
	PrefillState  int64
}

type Implementation struct {
	Name    string
	RunFunc func(numG int, config BenchConfig)
}

func runBenchmark(name, implementation, scenario string, numG int, runFunc func()) BenchmarkResult {
	runtime.GC()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	before := captureMemSnapshot()

	runFunc()

	time.Sleep(100 * time.Millisecond)

	runtime.GC()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	after := captureMemSnapshot()

	delta := calculateDelta(before, after)

	return BenchmarkResult{
		Name:           name,
		Implementation: implementation,
		Goroutines:     numG,
		Scenario:       scenario,
		Before:         before,
		After:          after,
		Delta:          delta,
	}
}

func generateTextReport(report ComparisonReport) {
	f, err := os.Create(fmt.Sprintf("%s.txt", report.Name))
	if err != nil {
		fmt.Printf("Error creating text report: %v\n", err)
		return
	}
	defer f.Close()

	separator := strings.Repeat("=", 100)
	lineSep := strings.Repeat("-", 150)

	fmt.Fprintf(f, "MEMORY BENCHMARK REPORT\n")
	fmt.Fprintf(f, "Generated: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(f, "%s", separator+"\n\n")

	scenarios := make(map[string][]BenchmarkResult)
	for _, result := range report.Results {
		scenarios[result.Scenario] = append(scenarios[result.Scenario], result)
	}

	implNames := make(map[string]bool)
	for _, result := range report.Results {
		implNames[result.Implementation] = true
	}

	for scenario, results := range scenarios {
		fmt.Fprintf(f, "\n>>> SCENARIO: %s\n", scenario)
		fmt.Fprintf(f, "%s", separator+"\n\n")

		byGoroutines := make(map[int][]BenchmarkResult)
		for _, r := range results {
			byGoroutines[r.Goroutines] = append(byGoroutines[r.Goroutines], r)
		}

		gCounts := make([]int, 0, len(byGoroutines))
		for g := range byGoroutines {
			gCounts = append(gCounts, g)
		}

		for i := 0; i < len(gCounts); i++ {
			for j := i + 1; j < len(gCounts); j++ {
				if gCounts[i] > gCounts[j] {
					gCounts[i], gCounts[j] = gCounts[j], gCounts[i]
				}
			}
		}

		for _, g := range gCounts {
			items := byGoroutines[g]
			fmt.Fprintf(f, "Goroutines: %d\n", g)
			fmt.Fprintf(f, "%s", lineSep+"\n")

			resultsByImpl := make(map[string]*BenchmarkResult)
			for i := range items {
				resultsByImpl[items[i].Implementation] = &items[i]
			}

			implList := make([]string, 0, len(resultsByImpl))
			for impl := range resultsByImpl {
				implList = append(implList, impl)
			}

			for i := 0; i < len(implList); i++ {
				for j := i + 1; j < len(implList); j++ {
					if implList[i] > implList[j] {
						implList[i], implList[j] = implList[j], implList[i]
					}
				}
			}

			if len(resultsByImpl) >= 2 {

				header := fmt.Sprintf("%-40s", "Metric")
				for _, impl := range implList {
					header += fmt.Sprintf(" | %-25s", impl)
				}
				fmt.Fprintf(f, "%s\n", header)
				fmt.Fprintf(f, "%s", lineSep+"\n")

				row := fmt.Sprintf("%-40s", "Alloc Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25s", formatBytes(r.Delta.AllocDelta))
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "HeapObjects Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25d", r.Delta.HeapObjectsDelta)
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "TotalAlloc Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25s", formatBytes(r.Delta.TotalAllocDelta))
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "Mallocs Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25d", r.Delta.MallocsDelta)
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "Frees Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25d", r.Delta.FreesDelta)
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "HeapInuse Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25s", formatBytes(r.Delta.HeapInuseDelta))
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "StackInuse Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25s", formatBytes(r.Delta.StackInuseDelta))
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "NumGC Delta")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						row += fmt.Sprintf(" | %25d", r.Delta.NumGCDelta)
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				row = fmt.Sprintf("%-40s", "Live Objects After GC")
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						live := int64(r.After.Mallocs) - int64(r.After.Frees)
						row += fmt.Sprintf(" | %25d", live)
					}
				}
				fmt.Fprintf(f, "%s\n", row)

				fmt.Fprintf(f, "\n%-40s | ", "Winner (Lowest Memory)")
				minAlloc := int64(1 << 62)
				winner := ""
				for _, impl := range implList {
					if r, ok := resultsByImpl[impl]; ok {
						if r.Delta.AllocDelta < minAlloc {
							minAlloc = r.Delta.AllocDelta
							winner = impl
						}
					}
				}
				fmt.Fprintf(f, "%s", winner)
				if winner != "" {

					for _, impl := range implList {
						if impl != winner {
							if r, ok := resultsByImpl[impl]; ok {
								diff := r.Delta.AllocDelta - minAlloc
								fmt.Fprintf(f, " (vs %s: %s more)", impl, formatBytes(diff))
							}
						}
					}
				}
				fmt.Fprintf(f, "\n")
			}

			fmt.Fprintf(f, "\n")
		}
	}

	fmt.Fprintf(f, "%s", "\n"+separator+"\n")
	fmt.Fprintf(f, "SUMMARY\n")
	fmt.Fprintf(f, "%s", separator+"\n\n")

	for scenario := range scenarios {
		fmt.Fprintf(f, "Scenario: %s\n", scenario)

		implSummary := make(map[string]struct {
			TotalAlloc int64
			AvgAlloc   int64
			Count      int
		})

		for _, result := range report.Results {
			if result.Scenario == scenario {
				s := implSummary[result.Implementation]
				s.TotalAlloc += result.Delta.AllocDelta
				s.Count++
				implSummary[result.Implementation] = s
			}
		}

		for impl, summary := range implSummary {
			avgAlloc := summary.TotalAlloc / int64(summary.Count)
			fmt.Fprintf(f, "  %s: Avg Alloc Delta = %s (across %d tests)\n",
				impl, formatBytes(avgAlloc), summary.Count)
		}
		fmt.Fprintf(f, "\n")
	}
}
