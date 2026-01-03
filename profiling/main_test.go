package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func BenchmarkConcatWithPlus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConcatWithPlus("Hello, ", "World!")
	}
}

func BenchmarkConcatWithArgsRange(b *testing.B) {
	strs := []string{"Hello, ", "World!"}
	for i := 0; i < b.N; i++ {
		ConcatWithArgsRange(strs)
	}
}
func BenchmarkConcatWithBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConcatWithBuilder("Hello, ", "World!")
	}
}
func BenchmarkConcatWithSprintf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConcatWithSprintf("Hello, ", "World!")
	}
}

func BenchmarkManyArgsRange(b *testing.B) {
	// strs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	strs := prepLongJSON()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConcatWithArgsRange(strs)
	}
}

func BenchmarkManyStringsBuilder(b *testing.B) {
	// strs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	strs := prepLongJSON()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var builder strings.Builder
		for _, s := range strs {
			builder.WriteString(s)
		}
		_ = builder.String()
	}
}

// func BenchmarkStringOps(b *testing.B) {
// 	b.Run("Plus", func(b *testing.B) {
// 		for i := 0; i < b.N; i++ {
// 			_ = ConcatWithPlus("a", "b")
// 		}
// 	})

// 	b.Run("Builder", func(b *testing.B) {
// 		for i := 0; i < b.N; i++ {
// 			_ = ConcatWithBuilder("a", "b")
// 		}
// 	})
// }

func BenchmarkWithMetrics(b *testing.B) {
	var bytesProcessed int64

	for i := 0; i < b.N; i++ {
		n := processData()
		bytesProcessed += int64(n)
	}

	// 5. REPORT METRIC - Custom metrics
	b.ReportMetric(float64(bytesProcessed)/float64(b.N), "bytes/op")

	// 6. REPORT ALLOCS - Manually report allocations if needed
	b.ReportAllocs()
}

func prepLongJSON() []string {
	testJson, err := json.Marshal(`{
  "id": 1,
  "title": "Go Application Monitoring",
  "tags": ["go", "application", "monitoring"],
  "style": "dark",
  "timezone": "browser",
  "panels": [
    {
      "id": 1,
      "title": "Memory Usage (MB)",
      "type": "timeseries",
      "targets": [
        {
          "expr": "go_memstats_alloc_bytes{job=\"chat-app\"} / 1024 / 1024",
          "legendFormat": "Allocated Memory",
          "refId": "A"
        },
        {
          "expr": "go_memstats_sys_bytes{job=\"chat-app\"} / 1024 / 1024",
          "legendFormat": "System Memory",
          "refId": "B"
        },
        {
          "expr": "go_memstats_heap_alloc_bytes{job=\"chat-app\"} / 1024 / 1024",
          "legendFormat": "Heap Allocated",
          "refId": "C"
        },
        {
          "expr": "go_memstats_heap_sys_bytes{job=\"chat-app\"} / 1024 / 1024",
          "legendFormat": "Heap System",
          "refId": "D"
        }
      ],
      "unit": "decbytes",
      "min": 0,
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 0
      }
    },
    {
      "id": 2,
      "title": "CPU Usage",
      "type": "timeseries",
      "targets": [
        {
          "expr": "rate(process_cpu_seconds_total{job=\"chat-app\"}[5m]) * 100",
          "legendFormat": "CPU Usage %",
          "refId": "A"
        }
      ],
      "unit": "percent",
      "min": 0,
      "max": 100,
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 12,
        "y": 0
      }
    },
    {
      "id": 3,
      "title": "Garbage Collection Duration (ms)",
      "type": "timeseries",
      "targets": [
        {
          "expr": "rate(go_gc_duration_seconds_sum{job=\"chat-app\"}[5m]) * 1000",
          "legendFormat": "GC Duration (avg)",
          "refId": "A"
        },
        {
          "expr": "go_gc_duration_seconds{job=\"chat-app\", quantile=\"0.5\"} * 1000",
          "legendFormat": "GC Duration (50th percentile)",
          "refId": "B"
        },
        {
          "expr": "go_gc_duration_seconds{job=\"chat-app\", quantile=\"0.95\"} * 1000",
          "legendFormat": "GC Duration (95th percentile)",
          "refId": "C"
        }
      ],
      "unit": "ms",
      "min": 0,
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 8
      }
    },
    {
      "id": 4,
      "title": "Garbage Collection Rate",
      "type": "timeseries",
      "targets": [
        {
          "expr": "rate(go_gc_duration_seconds_count{job=\"chat-app\"}[5m])",
          "legendFormat": "GC Rate (collections/sec)",
          "refId": "A"
        }
      ],
      "unit": "ops",
      "min": 0,
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 12,
        "y": 8
      }
    },
    {
      "id": 5,
      "title": "Goroutines",
      "type": "timeseries",
      "targets": [
        {
          "expr": "go_goroutines{job=\"chat-app\"}",
          "legendFormat": "Goroutines",
          "refId": "A"
        }
      ],
      "unit": "short",
      "min": 0,
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 16
      }
    },
    {
      "id": 6,
      "title": "Heap Objects (Count)",
      "type": "timeseries",
      "targets": [
        {
          "expr": "go_memstats_heap_objects{job=\"chat-app\"}",
          "legendFormat": "Heap Objects",
          "refId": "A"
        }
      ],
      "unit": "short",
      "min": 0,
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 12,
        "y": 16
      }
    },
    {
      "id": 7,
      "title": "Memory Allocation Rate (MB/s)",
      "type": "timeseries",
      "targets": [
        {
          "expr": "rate(go_memstats_alloc_bytes_total{job=\"chat-app\"}[5m]) / 1024 / 1024",
          "legendFormat": "Allocation Rate",
          "refId": "A"
        }
      ],
      "unit": "decbytes",
      "min": 0,
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 24
      }
    },
    {
      "id": 8,
      "title": "Memory Stats Summary",
      "type": "stat",
      "targets": [
        {
          "expr": "go_memstats_alloc_bytes{job=\"chat-app\"} / 1024 / 1024",
          "legendFormat": "Current Alloc (MB)",
          "refId": "A"
        },
        {
          "expr": "go_memstats_sys_bytes{job=\"chat-app\"} / 1024 / 1024",
          "legendFormat": "System Memory (MB)",
          "refId": "B"
        },
        {
          "expr": "go_goroutines{job=\"chat-app\"}",
          "legendFormat": "Goroutines",
          "refId": "C"
        },
        {
          "expr": "go_memstats_heap_objects{job=\"chat-app\"}",
          "legendFormat": "Heap Objects",
          "refId": "D"
        }
      ],
      "unit": "short",
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 12,
        "y": 24
      }
    }
  ],
  "time": {
    "from": "now-1h",
    "to": "now"
  },
  "timepicker": {},
  "templating": {
    "list": []
  },
  "annotations": {
    "list": []
  },
  "refresh": "5s",
  "schemaVersion": 27,
  "version": 0,
  "links": []
}
`)
	if err != nil {
		panic(err)
	}
	res := []string{}
	for _, r := range testJson {
		res = append(res, string(r))
	}

	return res
}
