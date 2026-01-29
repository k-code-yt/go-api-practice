package main

import (
	"fmt"
	"testing"
	"time"
)

func Benchmark_UpdateHeavy(t *testing.B) {
	cfg := NewTestConfig(200, 10, 10, 100, false)
	ps := NewPartitionState(cfg)
	ps.init()

	testDur := time.Second * 10

	<-time.After(testDur)
	fmt.Println("---TEST IS OVER---")
	ps.Cancel()
	<-ps.ctx.Done()
	fmt.Println("---EXIT TEST---")
}

func Benchmark_WriteWithLock(b *testing.B) {
	cfg := NewTestConfig(100, 50, 10, 100, false)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ps := NewPartitionState(cfg)
		ps.init()

		// Run for a fixed duration to accumulate state
		time.Sleep(5 * time.Second)

		ps.Cancel()
		<-ps.ExitCH
	}
}
