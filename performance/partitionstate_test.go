package main

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

var (
	cfg = NewTestConfig(
		1,      //IDX
		2000,   // commit,
		1000,   //update
		1500,   //append
		250,    //test
		100000, //updaterange
		"lock", //scenario
		false,  //debugMode
	)
)

func Benchmark_UpdateHeavy(t *testing.B) {
	ps := NewPartitionStateLock(cfg)
	ps.init()

	testDur := time.Second * 10

	<-time.After(testDur)
	fmt.Println("---TEST IS OVER---")
	ps.Cancel()
	<-ps.ctx.Done()
	fmt.Println("---EXIT TEST---")
}

func Benchmark_All(b *testing.B) {
	// gCounts := []int{4}
	gCounts := []int{2, 4, 8, 16, 32}

	for _, numGoroutines := range gCounts {
		b.Run(fmt.Sprintf("LOCK_G=%d", numGoroutines), func(b *testing.B) {

			opsPerG := b.N / numGoroutines
			if opsPerG == 0 {
				opsPerG = 1
			}
			wg := new(sync.WaitGroup)
			wg.Add(numGoroutines)

			cfg.appendDur = cfg.appendDur / time.Duration(opsPerG)
			cfg.updateDur = cfg.updateDur / time.Duration(opsPerG)
			cfg.commitDur = cfg.commitDur / time.Duration(opsPerG)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer wg.Done()
					ps := NewPartitionStateLock(cfg)
					ps.init()
					<-time.After(10 * time.Second)
					ps.Cancel()
				}()
			}
			wg.Wait()
		})

		b.Run(fmt.Sprintf("SYNCMAP_G=%d", numGoroutines), func(b *testing.B) {
			opsPerG := b.N / numGoroutines
			if opsPerG == 0 {
				opsPerG = 1
			}
			wg := new(sync.WaitGroup)
			wg.Add(numGoroutines)

			cfg.appendDur = cfg.appendDur / time.Duration(opsPerG)
			cfg.updateDur = cfg.updateDur / time.Duration(opsPerG)
			cfg.commitDur = cfg.commitDur / time.Duration(opsPerG)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer wg.Done()
					ps := NewPartitionStateSyncMap(cfg)
					ps.init()
					<-time.After(10 * time.Second)
					ps.Cancel()
				}()
			}
			wg.Wait()

		})
	}

}

func Benchmark_Lock_PS(b *testing.B) {
	opsPerG := b.N / numGoroutines
	if opsPerG == 0 {
		opsPerG = 1
	}
	fmt.Println(opsPerG)
	wg := new(sync.WaitGroup)
	wg.Add(numGoroutines)

	cfg.appendDur = cfg.appendDur / time.Duration(opsPerG)
	cfg.updateDur = cfg.updateDur / time.Duration(opsPerG)
	cfg.commitDur = cfg.commitDur / time.Duration(opsPerG)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ps := NewPartitionStateLock(cfg)
			ps.init()
			<-time.After(10 * time.Second)
			ps.Cancel()
		}()
	}
	wg.Wait()

}

func Benchmark_SyncMap_PS(b *testing.B) {
	b.Run("SM", func(b *testing.B) {
		opsPerG := b.N / numGoroutines
		if opsPerG == 0 {
			opsPerG = 1
		}
		fmt.Println(opsPerG)
		wg := new(sync.WaitGroup)
		wg.Add(numGoroutines)

		cfg.appendDur = cfg.appendDur / time.Duration(opsPerG)
		cfg.updateDur = cfg.updateDur / time.Duration(opsPerG)
		cfg.commitDur = cfg.commitDur / time.Duration(opsPerG)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				ps := NewPartitionStateSyncMap(cfg)
				ps.init()
				<-time.After(2 * time.Second)
				ps.Cancel()
			}()
		}
		wg.Wait()
	})
}

func Benchmark_TrackingLock(b *testing.B) {
	wg := &sync.WaitGroup{}
	numG := 8
	PrintMemStats("START")

	wg.Add(numG)
	for range numG {
		go func() {
			defer wg.Done()
			testCfg := NewTestConfig(1, 2, 1, 2, 250, 10000, "lock", false)
			ps := NewPartitionStateLock(testCfg)
			ps.init()

			<-time.After(5 * time.Second)
			ps.Cancel()
			<-ps.exit()

			// TODO -> investigate how to remove reference for GC collection
			// clean-up
			ps.MaxReceived = nil
			ps.cfg = nil
			ps.Mu.Lock()
			for k := range ps.State {
				delete(ps.State, k)
			}
			ps.Mu.Unlock()
			ps.Mu = nil
			ps.State = nil
		}()
	}
	wg.Wait()
	PrintMemStats("AFTER RUN")
	runtime.GC()
	PrintMemStats("AFTER GC")
}

func Benchmark_TrackingSM(b *testing.B) {
	wg := &sync.WaitGroup{}
	numG := 8
	PrintMemStats("START")

	wg.Add(numG)
	for range numG {
		go func() {
			defer wg.Done()
			testCfg := NewTestConfig(1, 2, 1, 2, 250, 10000, "lock", false)
			ps := NewPartitionStateSyncMap(testCfg)
			ps.init()

			<-time.After(5 * time.Second)
			ps.Cancel()
			<-ps.exit()

			// clean-up
			ps.MaxReceived = nil
			ps.cfg = nil
			ps.State.Range(func(key, value any) bool {
				ps.State.Delete(key)
				return true
			})
			ps.State = nil
		}()
	}
	wg.Wait()
	PrintMemStats("AFTER RUN")
	runtime.GC()
	PrintMemStats("AFTER GC")
}

func Benchmark_TrackingAll(b *testing.B) {
	numG := 16
	testDur := time.Duration(30)
	testCfg := NewTestConfig(1, 2, 2, 1, 250, 10000, "lock", false)

	b.Run(fmt.Sprintf("LOCK_%d\n", numG), func(b *testing.B) {

		wg := &sync.WaitGroup{}
		// PrintMemStats("START")

		wg.Add(numG)
		for range numG {
			go func() {
				defer wg.Done()

				ps := NewPartitionStateLock(testCfg)
				ps.init()

				<-time.After(testDur * time.Second)
				ps.Cancel()
				<-ps.exit()

				// clean-up
				ps.MaxReceived = nil
				ps.cfg = nil
				ps.Mu.Lock()
				for k := range ps.State {
					delete(ps.State, k)
				}
				ps.Mu.Unlock()
				ps.Mu = nil
				ps.State = nil
			}()
		}
		wg.Wait()
		PrintMemStats("AFTER RUN")
		runtime.GC()
		// PrintMemStats("AFTER GC")
	})

	b.Run(fmt.Sprintf("SM_%d\n", numG), func(b *testing.B) {
		wg := &sync.WaitGroup{}
		numG := 8
		// PrintMemStats("START")

		wg.Add(numG)
		for range numG {
			go func() {
				defer wg.Done()
				// testCfg := NewTestConfig(1, 2, 1, 2, 250, 10000, "syncmap", false)
				ps := NewPartitionStateSyncMap(testCfg)
				ps.init()

				<-time.After(testDur * time.Second)
				ps.Cancel()
				<-ps.exit()

				// clean-up
				ps.MaxReceived = nil
				ps.cfg = nil
				ps.State.Range(func(key, value any) bool {
					ps.State.Delete(key)
					return true
				})
				ps.State = nil
			}()
		}
		wg.Wait()
		PrintMemStats("AFTER RUN")
		runtime.GC()
		// PrintMemStats("AFTER GC")
	})
}
