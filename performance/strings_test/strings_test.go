package main

import (
	"fmt"
	"strings"
	"testing"
)

func Benchmark_Concat(b *testing.B) {
	str := "name"
	b.ReportAllocs()
	fmt.Println(b.N)
	for i := 0; i < b.N; i++ {
		str += str
	}
}

func Benchmark_Builder(b *testing.B) {
	builder := strings.Builder{}
	str := "name"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		builder.WriteString(str)
	}
}
