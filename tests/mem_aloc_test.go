package tests

import (
	"fmt"
	"testing"
)

var buffer []byte = make([]byte, 33)
var result string

func BenchmarkConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		result = string(buffer) + string(buffer)
	}
}

func TestBench(t *testing.T) {
	b := testing.Benchmark(BenchmarkConcat)
	fmt.Println(b.AllocsPerOp())
	fmt.Println(b.AllocedBytesPerOp())
}
