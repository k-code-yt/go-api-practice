package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-chans/internal/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-chans/internal/shared"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	findLatestCallsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "find_latest_calls_total",
		Help: "Total number of FindLatestToCommit calls",
	})

	findLatestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "find_latest_duration_seconds",
		Help:    "Duration of FindLatestToCommit calls",
		Buckets: prometheus.DefBuckets,
	})

	memoryAlloc = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_alloc_bytes",
		Help: "Current memory allocation",
	})

	goroutinesCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "goroutines_count",
		Help: "Number of goroutines",
	})

	// activeRequests atomic.Int64
)

type BenchmarkServer struct {
	ps       *consumer.PartitionState
	msgCount int
}

var (
	MsgCount = 1_000
)

func NewBenchmarkServer(msgCount int) *BenchmarkServer {
	if msgCount <= 0 {
		msgCount = MsgCount
	}

	ps := NewTestPartitionState(msgCount)

	return &BenchmarkServer{
		ps:       ps,
		msgCount: msgCount,
	}
}

func NewTestPartitionState(msgCount int) *consumer.PartitionState {
	topic := "test_topic"
	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	ps := consumer.NewPartitionState(tp)

	state := shared.MsgState_Success
	for i := range msgCount {
		offset := kafka.Offset(i)
		if i%5 == 0 {
			msg := consumer.NewUpdateStateMsg(offset, shared.MsgState_Pending)
			ps.UpdateStateCH <- msg
			continue
		}
		msg := consumer.NewUpdateStateMsg(offset, state)
		ps.UpdateStateCH <- msg
	}

	ps.MaxReceived.Offset = kafka.Offset(msgCount)

	return ps

}

func (s *BenchmarkServer) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	qp := r.URL.Query()
	itersStr := qp.Get("iters")
	iters := 1_000_000
	var err error
	if itersStr != "" {
		iters, err = strconv.Atoi(itersStr)
		if err != nil {
			iters = 1_000_000
		}
	}
	state := shared.MsgState_Success
	for range iters {
		findLatestCallsTotal.Inc()
		s.ps.FindLatestToCommitReqCH <- struct{}{}
		offset := <-s.ps.FindLatestToCommitRespCH

		msg := consumer.NewUpdateStateMsg(offset, state)
		s.ps.UpdateStateCH <- msg
	}

	durr := time.Since(start)
	findLatestDuration.Observe(durr.Seconds())
	resp := map[string]any{
		"durr":       durr.Seconds(),
		"iterations": iters,
	}
	s.writeJSON(w, resp)

}
func (s *BenchmarkServer) handleReset(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	msgCountStr := qp.Get("msgCount")
	msgCount := 1_000_000
	var err error
	if msgCountStr != "" {
		msgCount, err = strconv.Atoi(msgCountStr)
		if err != nil {
			msgCount = 1_000_000
		}
	}
	MsgCount = msgCount
	s.ps.Cancel()
	s.ps = nil
	fmt.Printf("cleaned oldPS = %v,", s.ps)

	s.msgCount = msgCount
	newPS := NewTestPartitionState(msgCount)
	s.ps = newPS

	resp := map[string]any{
		"status":      http.StatusOK,
		"newMsgCount": msgCount,
	}
	s.writeJSON(w, resp)

}

func (s *BenchmarkServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	m := updateMemStats()
	resp := map[string]any{
		"status":       http.StatusOK,
		"mem_alloc_mb": s.convertToMB(m.Alloc),
		"mem_sys_mb":   s.convertToMB(m.Sys),
		"goroutines":   runtime.NumGoroutine(),
	}
	s.writeJSON(w, resp)
}

func (s *BenchmarkServer) convertToMB(val uint64) float64 {
	return float64(val) / 1024 / 1024
}

func (s *BenchmarkServer) writeJSON(w http.ResponseWriter, resp map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func updateMemStats() *runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryAlloc.Set(float64(m.Alloc))
	goroutinesCount.Set(float64(runtime.NumGoroutine()))
	return &m
}

func main() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateMemStats()
			fmt.Println("Emmiting metrics")
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s := NewBenchmarkServer(-1)
	defer s.ps.Cancel()

	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/run-benchmark", s.handleBenchmark)
	http.HandleFunc("/reset", s.handleReset)
	http.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	fmt.Printf("Server is running in %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
