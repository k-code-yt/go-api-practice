package main

import (
	"fmt"
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
