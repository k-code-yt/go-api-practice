package main

import (
	"fmt"
	"testing"
	"time"
)

var (
	cfg = NewTestConfig(1000, 100, 250, 100, false)
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

func Benchmark_WriteWithLock(b *testing.B) {
	exitCH := make(chan struct{})
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		go func() {
			ps := NewPartitionStateLock(cfg)
			ps.init()
			<-exitCH
			ps.Cancel()
		}()

		go func(i int) {
			time.Sleep(5 * time.Second)
			if i == 0 {
				close(exitCH)
			}
		}(i)
	}
	<-exitCH
}

func Benchmark_WriteWithSyncMap(b *testing.B) {
	exitCH := make(chan struct{})
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		go func() {
			ps := NewPartitionStateSyncMap(cfg)
			ps.init()
			<-exitCH
			ps.Cancel()
		}()

		go func(i int) {
			time.Sleep(5 * time.Second)
			if i == 0 {
				close(exitCH)
			}
		}(i)
	}
	<-exitCH

}
