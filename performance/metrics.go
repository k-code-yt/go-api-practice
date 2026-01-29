package main

import (
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// var (
// 	goroutinesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
// 		Name: "app_goroutines_count",
// 		Help: "Number of goroutines currently running",
// 	})

// 	heapAllocGauge = prometheus.NewGauge(prometheus.GaugeOpts{
// 		Name: "app_heap_alloc_mb",
// 		Help: "Bytes of allocated heap objects in MB",
// 	})

// 	sysMemoryGauge = prometheus.NewGauge(prometheus.GaugeOpts{
// 		Name: "app_sys_memory_mb",
// 		Help: "Total memory obtained from OS in MB",
// 	})

// 	cpuUsageGauge = prometheus.NewGauge(prometheus.GaugeOpts{
// 		Name: "app_cpu_usage_percent",
// 		Help: "CPU usage percentage",
// 	})

// 	rssMemoryGauge = prometheus.NewGauge(prometheus.GaugeOpts{
// 		Name: "app_rss_memory_bytes",
// 		Help: "Resident set size memory",
// 	})
// )

func init() {
	prometheus.MustRegister(collectors.NewGoCollector())
	prometheus.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

func collectMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// goroutinesGauge.Set(float64(runtime.NumGoroutine()))
	// heapAllocGauge.Set(float64(m.HeapAlloc))
	// heapInuseGauge.Set(float64(m.HeapInuse))

	// CPU usage (simple approximation)
	// cpuUsageGauge.Set(float64(m.GCCPUFraction) * 100)
}

func startMetricsServer() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	go func() {
		for {
			collectMetrics()
			time.Sleep(1 * time.Second)
		}
	}()
}
