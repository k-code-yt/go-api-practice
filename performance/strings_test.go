package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const iters = 15

func runStringsConcat(b *testing.B, numG int, config BenchConfig) {
	fmt.Println(b.N)
	str := "name"
	b.ReportAllocs()
	time.Sleep(time.Second)
	for range iters {
		str += str
	}
}

func runStringsBuilder(b *testing.B, numG int, config BenchConfig) {
	str := []byte("name")
	strByteLen := len(str)
	builder := strings.Builder{}
	time.Sleep(time.Second)
	builder.Grow(iters * strByteLen)

	b.ReportAllocs()
	for i := 0; i < iters; i++ {
		for _, b := range str {
			builder.WriteByte(b)
		}
	}

}

func Benchmark_Concat(b *testing.B) {
	impl := Implementation{
		Name:    "Str-concat",
		RunFunc: runStringsConcat,
	}

	runBenchmark(
		fmt.Sprintf("%s_G%d_%s", impl.Name, GoroutineCounts, impl.Name),
		impl.Name,
		impl.Name,
		GoroutineCounts,
		func() { impl.RunFunc(b, GoroutineCounts, AppedHeavyConfig) },
	)

}

func Benchmark_Builder(b *testing.B) {
	impl := Implementation{
		Name:    "Str-builder",
		RunFunc: runStringsBuilder,
	}
	runBenchmark(
		fmt.Sprintf("%s_G%d_%s", impl.Name, GoroutineCounts, impl.Name),
		impl.Name,
		impl.Name,
		GoroutineCounts,
		func() { impl.RunFunc(b, GoroutineCounts, AppedHeavyConfig) },
	)
}
