package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	ResultsFolder     = "temp"
	ResultsFileSuffix = "json"
)

type MemSnapshot struct {
	Alloc         uint64    `json:"Alloc"`
	TotalAlloc    uint64    `json:"TotalAlloc"`
	Sys           uint64    `json:"Sys"`
	HeapObjects   uint64    `json:"HeapObjects"`
	HeapInuse     uint64    `json:"HeapInuse"`
	HeapIdle      uint64    `json:"HeapIdle"`
	StackInuse    uint64    `json:"StackInuse"`
	Mallocs       uint64    `json:"Mallocs"`
	Frees         uint64    `json:"Frees"`
	NumGC         uint32    `json:"NumGC"`
	GCCPUFraction float64   `json:"GCCPUFraction"`
	Timestamp     time.Time `json:"Timestamp"`
}

type MemDelta struct {
	AllocDelta         int64         `json:"AllocDelta"`
	TotalAllocDelta    int64         `json:"TotalAllocDelta"`
	SysDelta           int64         `json:"SysDelta"`
	HeapObjectsDelta   int64         `json:"HeapObjectsDelta"`
	HeapInuseDelta     int64         `json:"HeapInuseDelta"`
	HeapIdleDelta      int64         `json:"HeapIdleDelta"`
	StackInuseDelta    int64         `json:"StackInuseDelta"`
	MallocsDelta       int64         `json:"MallocsDelta"`
	FreesDelta         int64         `json:"FreesDelta"`
	NumGCDelta         int32         `json:"NumGCDelta"`
	GCCPUFractionDelta float64       `json:"GCCPUFractionDelta"`
	Duration           time.Duration `json:"Duration"`
}

type BenchmarkResult struct {
	Name           string `json:"Name"`
	Implementation string `json:"Implementation"`
	Goroutines     int    `json:"Goroutines"`
	Scenario       string `json:"Scenario"`
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
	RunFunc func(b *testing.B, numG int, config BenchConfig)
}

func Benchmark_ParseResults(b *testing.B) {
	files, err := os.ReadDir(ResultsFolder)
	if err != nil {
		panic(err)
	}

	toProcess := []os.DirEntry{}
	for _, file := range files {
		if hasSuffix(file.Name(), ResultsFileSuffix) {
			toProcess = append(toProcess, file)
		}
	}

	if len(toProcess) == 0 {
		return
	}

	results := make([]BenchmarkResult, len(toProcess))
	for idx, file := range toProcess {
		fPath := fmt.Sprintf("%s/%s", ResultsFolder, file.Name())
		b, err := os.ReadFile(fPath)
		if err != nil {
			panic(err)
		}

		result := BenchmarkResult{}
		err = json.Unmarshal(b, &result)
		if err != nil {
			panic(err)
		}
		results[idx] = result
	}

	time.Sleep(time.Second)

	cr := ComparisonReport{
		Name:      fmt.Sprintf("%d_%s_bench_result", results[0].Goroutines, results[0].Scenario),
		Timestamp: time.Now(),
		Results:   results,
	}

	generateTextReport(cr)

	for _, file := range toProcess {
		fPath := fmt.Sprintf("%s/%s", ResultsFolder, file.Name())

		err = os.Remove(fPath)
		if err != nil {
			panic(err)
		}
	}

}

func hasSuffix(path string, suffix string) bool {
	return strings.HasSuffix(path, suffix)
}

func runBenchmark(name, implementation, scenario string, numG int, runFunc func()) string {
	before := captureMemSnapshot()

	runFunc()

	after := captureMemSnapshot()
	delta := calculateDelta(before, after)

	br := BenchmarkResult{
		Name:           name,
		Implementation: implementation,
		Goroutines:     numG,
		Scenario:       scenario,
		Before:         before,
		After:          after,
		Delta:          delta,
	}

	path := fmt.Sprintf("%s/%s_%s_bench_result.%s", ResultsFolder, br.Name, br.Implementation, ResultsFileSuffix)
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(br)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	return path
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
