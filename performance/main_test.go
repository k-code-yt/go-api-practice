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
