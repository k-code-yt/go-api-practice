package main

// import (
// 	"fmt"
// 	"net/http"
// 	"runtime"
// 	"sync"
// 	"time"

// 	"github.com/prometheus/client_golang/prometheus"
// 	"github.com/prometheus/client_golang/prometheus/collectors"
// 	"github.com/prometheus/client_golang/prometheus/promhttp"
// )

// var (
// 	metricsOnce sync.Once
// 	registry    *prometheus.Registry
// )

// func collectMetrics() {
// 	var m runtime.MemStats
// 	runtime.ReadMemStats(&m)
// }

// func startMetricsServer() {
// 	metricsOnce.Do(func() {
// 		// Create a new registry
// 		registry = prometheus.NewRegistry()

// 		// Register collectors with the custom registry
// 		registry.MustRegister(collectors.NewGoCollector())
// 		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

// 		metricsPort := ":2112"

// 		// Start metrics HTTP server in a goroutine
// 		go func() {
// 			// Use the custom registry
// 			http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
// 			fmt.Printf("Metrics server is listening on %s\n", metricsPort)
// 			if err := http.ListenAndServe(metricsPort, nil); err != nil {
// 				fmt.Printf("Metrics server error: %v\n", err)
// 			}
// 		}()

// 		// Start metrics collection in a goroutine
// 		go func() {
// 			ticker := time.NewTicker(1 * time.Second)
// 			defer ticker.Stop()

// 			for range ticker.C {
// 				collectMetrics()
// 			}
// 		}()
// 	})
// }
