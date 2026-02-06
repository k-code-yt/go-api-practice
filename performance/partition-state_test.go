package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	PrefillState = 1_000_000
	TestDur      = 30 * time.Second
)

var (
	ReadHeavyConfig = BenchConfig{
		CommitDur:    10,
		UpdateDur:    1000,
		AppendDur:    1000,
		UpdateRange:  100000,
		TestDur:      TestDur,
		PrefillState: PrefillState,
	}

	AppedHeavyConfig = BenchConfig{
		CommitDur:    50,
		UpdateDur:    100,
		AppendDur:    10,
		UpdateRange:  10000,
		TestDur:      TestDur,
		PrefillState: PrefillState,
	}

	WriteHeavyConfig = BenchConfig{
		CommitDur:    5,
		UpdateDur:    2,
		AppendDur:    50,
		UpdateRange:  10000,
		TestDur:      TestDur,
		PrefillState: PrefillState,
	}

	MixedConfig = BenchConfig{
		CommitDur:    20,
		UpdateDur:    10,
		AppendDur:    15,
		UpdateRange:  10000,
		TestDur:      TestDur,
		PrefillState: PrefillState,
	}
)

func runLockBenchmark(numG int, config BenchConfig) {
	wg := &sync.WaitGroup{}
	wg.Add(numG)

	for range numG {
		go func() {
			defer wg.Done()
			testCfg := NewTestConfig(1, config.CommitDur, config.UpdateDur, config.AppendDur, 250, config.UpdateRange, "lock", numG, config.PrefillState, false)
			ps := NewPartitionStateLock(testCfg)
			ps.init()

			<-time.After(config.TestDur)

			ps.Cancel()
			ps.wg.Wait()

			// Cleanup
			ps.Mu.Lock()
			for k := range ps.State {
				delete(ps.State, k)
			}
			ps.Mu.Unlock()
		}()
	}

	wg.Wait()
}

func runSyncMapBenchmark(numG int, config BenchConfig) {
	wg := &sync.WaitGroup{}
	wg.Add(numG)

	for range numG {
		go func() {
			defer wg.Done()
			testCfg := NewTestConfig(1, config.CommitDur, config.UpdateDur, config.AppendDur, 250, config.UpdateRange, "syncmap", numG, config.PrefillState, false)
			ps := NewPartitionStateSyncMap(testCfg)
			ps.init()

			<-time.After(config.TestDur)

			ps.Cancel()
			ps.wg.Wait()

			ps.State.Range(func(key, value interface{}) bool {
				ps.State.Delete(key)
				return true
			})
		}()
	}

	wg.Wait()
}

func runChansBenchmark(numG int, config BenchConfig) {
	wg := &sync.WaitGroup{}
	wg.Add(numG)

	for range numG {
		go func() {
			defer wg.Done()
			testCfg := NewTestConfig(1, config.CommitDur, config.UpdateDur, config.AppendDur, 250, config.UpdateRange, "chans", numG, config.PrefillState, false)
			ps := NewPartitionStateChans(testCfg)
			ps.init()

			<-time.After(config.TestDur)

			ps.Cancel()
			ps.wg.Wait()

			for k := range ps.State {
				delete(ps.State, k)
			}
		}()
	}

	wg.Wait()
}

func Benchmark_ComprehensiveComparison(b *testing.B) {
	report := ComparisonReport{
		Timestamp: time.Now(),
		Results:   make([]BenchmarkResult, 0),
	}

	implementations := []Implementation{
		{Name: "Lock", RunFunc: runLockBenchmark},
		{Name: "Chans", RunFunc: runChansBenchmark},
		{Name: "SyncMap", RunFunc: runSyncMapBenchmark},
	}

	goroutineCounts := []int{4}
	// goroutineCounts := []int{4, 8, 16, 32}
	scenarios := map[string]BenchConfig{
		"ReadHeavy": ReadHeavyConfig,
		// "AppendHeavy": AppedHeavyConfig,
		// "WriteHeavy": WriteHeavyConfig,
		// "Mixed":      MixedConfig,
	}

	separator := strings.Repeat("=", 80)
	fmt.Println("\n" + separator)
	fmt.Println("COMPREHENSIVE MEMORY BENCHMARK")
	fmt.Println(separator + "\n")

	for scenario, config := range scenarios {
		fmt.Printf("\n>>> Testing Scenario: %s\n", scenario)
		fmt.Printf("    Config: Commit=%dms, Update=%dms, Append=%dms\n\n",
			config.CommitDur, config.UpdateDur, config.AppendDur)

		for _, numG := range goroutineCounts {
			for _, impl := range implementations {
				fmt.Printf("  Running %s with %d goroutines... ", impl.Name, numG)
				result := runBenchmark(
					fmt.Sprintf("%s_G%d_%s", scenario, numG, impl.Name),
					impl.Name,
					scenario,
					numG,
					func() { impl.RunFunc(numG, config) },
				)
				report.Results = append(report.Results, result)
				fmt.Printf("✓ (Alloc Delta: %s, Objects: %+d)\n",
					formatBytes(result.Delta.AllocDelta),
					result.Delta.HeapObjectsDelta)
			}
		}
	}

	name := ""
	for _, imp := range implementations {
		name += imp.Name
	}
	report.Name = name

	generateTextReport(report)

	fmt.Println("\n" + separator)
	fmt.Printf("Report generated: %s.txt\n", name)
	fmt.Println(separator + "\n")
}
func Benchmark_ComprehensiveSeparate(b *testing.B) {
	report := ComparisonReport{
		Timestamp: time.Now(),
		Results:   make([]BenchmarkResult, 0),
	}

	implementations := []Implementation{
		{Name: "SyncMap", RunFunc: runSyncMapBenchmark},
		{Name: "Chans", RunFunc: runChansBenchmark},
		{Name: "Lock", RunFunc: runLockBenchmark},
	}

	goroutineCounts := []int{4}
	scenarios := map[string]BenchConfig{
		"ReadHeavy": ReadHeavyConfig,
	}

	separator := strings.Repeat("=", 80)
	fmt.Println("\n" + separator)
	fmt.Println("COMPREHENSIVE MEMORY BENCHMARK")
	fmt.Println(separator + "\n")

	b.Run(implementations[0].Name, func(b *testing.B) {
		impl := implementations[0]
		for scenario, config := range scenarios {
			fmt.Printf("\n>>> Testing Scenario: %s\n", scenario)
			fmt.Printf("    Config: Commit=%dms, Update=%dms, Append=%dms\n\n",
				config.CommitDur, config.UpdateDur, config.AppendDur)

			for _, numG := range goroutineCounts {
				fmt.Printf("  Running %s with %d goroutines... ", impl.Name, numG)
				result := runBenchmark(
					fmt.Sprintf("%s_G%d_%s", scenario, numG, impl.Name),
					impl.Name,
					scenario,
					numG,
					func() { impl.RunFunc(numG, config) },
				)
				report.Results = append(report.Results, result)
				fmt.Printf("✓ (Alloc Delta: %s, Objects: %+d)\n",
					formatBytes(result.Delta.AllocDelta),
					result.Delta.HeapObjectsDelta)
			}
		}
	})

	b.Run(implementations[1].Name, func(b *testing.B) {
		impl := implementations[1]
		for scenario, config := range scenarios {
			fmt.Printf("\n>>> Testing Scenario: %s\n", scenario)
			fmt.Printf("    Config: Commit=%dms, Update=%dms, Append=%dms\n\n",
				config.CommitDur, config.UpdateDur, config.AppendDur)

			for _, numG := range goroutineCounts {
				fmt.Printf("  Running %s with %d goroutines... ", impl.Name, numG)
				result := runBenchmark(
					fmt.Sprintf("%s_G%d_%s", scenario, numG, impl.Name),
					impl.Name,
					scenario,
					numG,
					func() { impl.RunFunc(numG, config) },
				)
				report.Results = append(report.Results, result)
				fmt.Printf("✓ (Alloc Delta: %s, Objects: %+d)\n",
					formatBytes(result.Delta.AllocDelta),
					result.Delta.HeapObjectsDelta)
			}
		}
	})

	b.Run(implementations[2].Name, func(b *testing.B) {
		impl := implementations[2]
		for scenario, config := range scenarios {
			fmt.Printf("\n>>> Testing Scenario: %s\n", scenario)
			fmt.Printf("    Config: Commit=%dms, Update=%dms, Append=%dms\n\n",
				config.CommitDur, config.UpdateDur, config.AppendDur)

			for _, numG := range goroutineCounts {
				fmt.Printf("  Running %s with %d goroutines... ", impl.Name, numG)
				result := runBenchmark(
					fmt.Sprintf("%s_G%d_%s", scenario, numG, impl.Name),
					impl.Name,
					scenario,
					numG,
					func() { impl.RunFunc(numG, config) },
				)
				report.Results = append(report.Results, result)
				fmt.Printf("✓ (Alloc Delta: %s, Objects: %+d)\n",
					formatBytes(result.Delta.AllocDelta),
					result.Delta.HeapObjectsDelta)
			}
		}
	})

	name := ""
	for _, imp := range implementations {
		name += imp.Name
	}
	report.Name = name

	generateTextReport(report)

	fmt.Println("\n" + separator)
	fmt.Printf("Report generated: %s.txt\n", name)
	fmt.Println(separator + "\n")
}
