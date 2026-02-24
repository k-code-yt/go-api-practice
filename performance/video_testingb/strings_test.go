package main

import (
	"runtime"
	"strings"
	"testing"
)

func BenchmarkStrConcat(b *testing.B) {
	b.ReportAllocs()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < b.N; i++ {
		result := ""
		for _, h := range HtmlParts {
			result += h
		}
		_ = result
	}
	runtime.ReadMemStats(&after)
	totMem := float64((after.TotalAlloc - before.TotalAlloc) / (1024 * 1024))
	b.ReportMetric(totMem, "MB-totals")
}

func BenchmarkStrBuilderNoGrow(b *testing.B) {
	b.ReportAllocs()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		for _, h := range HtmlParts {
			sb.WriteString(h)
		}
		_ = sb.String()
	}
	runtime.ReadMemStats(&after)
	totMem := float64((after.TotalAlloc - before.TotalAlloc) / (1024 * 1024))
	b.ReportMetric(totMem, "MB-totals")
}

func BenchmarkStrBuilderGrow(b *testing.B) {
	b.ReportAllocs()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	total := 0
	for _, h := range HtmlParts {
		total += len(h)
	}
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.Grow(total)
		for _, h := range HtmlParts {
			sb.WriteString(h)
		}
		_ = sb.String()
	}
	runtime.ReadMemStats(&after)
	totMem := float64((after.TotalAlloc - before.TotalAlloc) / (1024 * 1024))
	b.ReportMetric(totMem, "MB-totals")
}
