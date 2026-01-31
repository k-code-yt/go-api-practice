package main

import (
	"fmt"
	"testing"
	"time"
)

var (
	cfg = NewTestConfig(
		1,      //IDX
		1000,   // commit,
		100,    //update
		150,    //append
		250,    //test
		100,    //updaterange
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
	b.Run("LOCK", func(b *testing.B) {
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
	})

	b.Run("SYNC_MAP", func(b *testing.B) {
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

	})
}
func Benchmark_Lock(b *testing.B) {
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

func Benchmark_SyncMap(b *testing.B) {

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
