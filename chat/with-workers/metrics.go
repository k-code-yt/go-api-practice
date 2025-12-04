package withworkers

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)
)

func registerMetrics() {
	prometheus.MustRegister(httpRequests)
	prometheus.MustRegister(httpDuration)
	prometheus.MustRegister(activeConnections)
}

func prometheusMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		activeConnections.Inc()
		defer activeConnections.Dec()

		next(w, r)

		duration := time.Since(start).Seconds()
		httpDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		httpRequests.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "alloc": %d, "sys": %d, "num_gc": %d}`,
		m.Alloc, m.Sys, m.NumGC)
}

func initProm() {
	registerMetrics()
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "2112"
	}

	http.HandleFunc("/health", prometheusMiddleware(healthHandler))
	http.Handle("/metrics", promhttp.Handler())
}
