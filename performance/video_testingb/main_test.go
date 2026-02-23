package main

import (
	"runtime"
	"testing"
)

func Benchmark_MathPow(b *testing.B) {
	b.ReportAllocs()
	sqr := 0.0
	for i := 0.0; i < float64(b.N); i++ {
		sqr = i * i
	}
	_ = sqr

}

func Benchmark_Slice(b *testing.B) {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	b.ReportAllocs()
	sl := []int{}
	for i := 0; i < b.N; i++ {
		sl = append(sl, i)
	}
	runtime.ReadMemStats(&after)

	alloctedBytes := float64(after.TotalAlloc-before.TotalAlloc) / float64(b.N)
	alloctedObj := float64(after.HeapObjects-before.HeapObjects) / float64(b.N)

	b.ReportMetric(alloctedBytes, "bytes/op-custom")
	b.ReportMetric(alloctedObj, "obj/op-custom")
}

func Benchmark_Reset(b *testing.B) {
	b.ReportAllocs()
	// Pre-setup
	data := make([]int, 1000000)
	for i := range data {
		data[i] = i
	}
	// b.ResetTimer()
	// actual tets
	for i := 0; i < b.N; i++ {
		sum := 0
		for _, v := range data {
			sum += v
		}
	}
}

func Benchmark_Loop(b *testing.B) {
	b.ReportAllocs()
	// Pre-setup
	data := make([]int, 1000000)
	for i := range data {
		data[i] = i
	}
	// actual tets
	for b.Loop() {
		sum := 0
		for _, v := range data {
			sum += v
		}
	}
}
