package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka/internal/consumer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	operationsTotal   *prometheus.CounterVec
	operationDuration *prometheus.HistogramVec
	memoryAllocMB     prometheus.Gauge
	goroutinesCount   prometheus.Gauge
	currentScenario   *prometheus.GaugeVec
)

type BenchmarkServer struct {
	kc         *consumer.KafkaConsumer
	msgCount   int
	nextOffset kafka.Offset
	mu         sync.RWMutex
}

func NewBenchmarkServer(msgCount int) *BenchmarkServer {
	if msgCount <= 0 {
		msgCount = 1000
	}

	topic := "test_topic"
	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	kc := consumer.NewTestKafkaConsumer(topic, tp)
	consumer.NewTestPartitionState(kc, tp, msgCount)

	return &BenchmarkServer{
		kc:         kc,
		msgCount:   msgCount,
		nextOffset: kafka.Offset(msgCount),
	}
}
func (s *BenchmarkServer) enableProm() {
	operationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "operations_total",
			Help: "Total number of operations by type",
		},
		[]string{"type", "scenario"},
	)

	operationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "operation_duration_seconds",
			Help:    "Duration of operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"scenario"},
	)

	memoryAllocMB = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_alloc_mb",
		Help: "Current memory allocation in MB",
	})

	goroutinesCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "goroutines_count",
		Help: "Number of goroutines",
	})

	currentScenario = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "current_scenario",
			Help: "Currently running scenario",
		},
		[]string{"name"},
	)
}

// Scenario 1: Read-heavy (80% read, 20% write)
func (s *BenchmarkServer) handleReadHeavy(w http.ResponseWriter, r *http.Request) {
	iters := s.getIters(r)
	scenario := "read-heavy"

	currentScenario.WithLabelValues(scenario).Set(1)
	defer currentScenario.WithLabelValues(scenario).Set(0)

	start := time.Now()
	ps, _ := s.kc.GetPartitionState(0)

	for i := 0; i < iters; i++ {
		if i%10 < 8 {
			randomOffset := kafka.Offset(rand.Intn(10000))
			_, _ = ps.ReadOffset(randomOffset)
			operationsTotal.WithLabelValues("read", scenario).Inc()
		} else {
			ps.Mu.RLock()
			randomOffset := kafka.Offset(rand.Intn(s.msgCount))
			ps.Mu.RUnlock()

			tp := &kafka.TopicPartition{
				Topic:     ps.MaxReceived.Topic,
				Partition: 0,
				Offset:    randomOffset,
			}

			s.kc.UpdateState(tp, consumer.MsgState_Success)
			operationsTotal.WithLabelValues("write", scenario).Inc()
		}
	}

	duration := time.Since(start)
	operationDuration.WithLabelValues(scenario).Observe(duration.Seconds())

	s.writeResponse(w, scenario, iters, duration)
}

// Scenario 2: Balanced (50% read, 50% write)
func (s *BenchmarkServer) handleBalanced(w http.ResponseWriter, r *http.Request) {
	iters := s.getIters(r)
	scenario := "balanced"

	currentScenario.WithLabelValues(scenario).Set(1)
	defer currentScenario.WithLabelValues(scenario).Set(0)

	start := time.Now()
	ps, _ := s.kc.GetPartitionState(0)

	for i := 0; i < iters; i++ {
		if i%2 == 0 {
			randomOffset := kafka.Offset(rand.Intn(10000))
			_, _ = ps.ReadOffset(randomOffset)
			operationsTotal.WithLabelValues("read", scenario).Inc()
		} else {
			ps.Mu.RLock()
			randomOffset := kafka.Offset(rand.Intn(s.msgCount))
			ps.Mu.RUnlock()

			tp := &kafka.TopicPartition{
				Topic:     ps.MaxReceived.Topic,
				Partition: 0,
				Offset:    randomOffset,
			}

			state := consumer.MsgState_Pending
			if rand.Intn(5) != 0 {
				state = consumer.MsgState_Success
			}
			s.kc.UpdateState(tp, state)
			operationsTotal.WithLabelValues("write", scenario).Inc()
		}
	}

	duration := time.Since(start)
	operationDuration.WithLabelValues(scenario).Observe(duration.Seconds())

	s.writeResponse(w, scenario, iters, duration)
}

// Scenario 3: Kafka simulation
func (s *BenchmarkServer) handleKafkaSim(w http.ResponseWriter, r *http.Request) {
	iters := s.getIters(r)
	scenario := "kafka-sim"

	currentScenario.WithLabelValues(scenario).Set(1)
	defer currentScenario.WithLabelValues(scenario).Set(0)

	start := time.Now()
	ps, _ := s.kc.GetPartitionState(0)

	for i := range iters {
		s.mu.Lock()
		newOffset := s.nextOffset
		s.nextOffset++
		s.mu.Unlock()

		tp := &kafka.TopicPartition{
			Topic:     ps.MaxReceived.Topic,
			Partition: 0,
			Offset:    newOffset,
		}

		s.kc.UpdateState(tp, consumer.MsgState_Pending)

		ps.Mu.Lock()
		if newOffset > ps.MaxReceived.Offset {
			ps.MaxReceived.Offset = newOffset
		}
		ps.Mu.Unlock()

		operationsTotal.WithLabelValues("write", scenario).Inc()

		result, err := ps.FindLatestToCommit()
		if err == nil && result != nil {
			operationsTotal.WithLabelValues("read", scenario).Inc()
			ps.Mu.Lock()
			ps.LastCommited = result.Offset
			ps.Mu.Unlock()

			if i%3 == 0 {
				for offset := result.Offset - kafka.Offset(rand.Intn(10)); offset < result.Offset; offset++ {
					tp := &kafka.TopicPartition{
						Topic:     ps.MaxReceived.Topic,
						Partition: 0,
						Offset:    offset,
					}
					s.kc.UpdateState(tp, consumer.MsgState_Success)
					operationsTotal.WithLabelValues("update", scenario).Inc()
				}
			}
		}
	}

	duration := time.Since(start)
	operationDuration.WithLabelValues(scenario).Observe(duration.Seconds())

	s.writeResponse(w, scenario, iters, duration)
}

func (s *BenchmarkServer) handleReset(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgCount := s.getMsgCount(r)

	_, err := s.kc.ResetPartitionState(0, msgCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.msgCount = msgCount
	s.nextOffset = kafka.Offset(msgCount)

	log.Printf("Reset complete. New msgCount: %d", msgCount)

	s.writeJSON(w, map[string]interface{}{
		"status":        "reset_complete",
		"new_msg_count": msgCount,
		"next_offset":   s.nextOffset,
	})
}

func (s *BenchmarkServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	m := updateMemStats()

	ps, _ := s.kc.GetPartitionState(0)
	stateSize := 0
	if ps != nil {
		ps.Mu.RLock()
		stateSize = len(ps.State)
		ps.Mu.RUnlock()
	}

	s.writeJSON(w, map[string]interface{}{
		"status":       "healthy",
		"mem_alloc_mb": float64(m.Alloc) / 1024 / 1024,
		"mem_sys_mb":   float64(m.Sys) / 1024 / 1024,
		"goroutines":   runtime.NumGoroutine(),
		"msg_count":    s.msgCount,
		"next_offset":  s.nextOffset,
		"state_size":   stateSize,
	})
}

func (s *BenchmarkServer) getIters(r *http.Request) int {
	itersStr := r.URL.Query().Get("iters")
	if itersStr == "" {
		return 10000
	}
	iters, err := strconv.Atoi(itersStr)
	if err != nil {
		return 10000
	}
	return iters
}

func (s *BenchmarkServer) getMsgCount(r *http.Request) int {
	msgCountStr := r.URL.Query().Get("msgCount")
	if msgCountStr == "" {
		return 1000
	}
	msgCount, err := strconv.Atoi(msgCountStr)
	if err != nil {
		return 1000
	}
	return msgCount
}

func (s *BenchmarkServer) writeResponse(w http.ResponseWriter, scenario string, iters int, duration time.Duration) {
	ps, _ := s.kc.GetPartitionState(0)
	stateSize := 0
	lastCommited := int64(0)
	maxReceived := int64(0)

	if ps != nil {
		ps.Mu.RLock()
		stateSize = len(ps.State)
		lastCommited = int64(ps.LastCommited)
		maxReceived = int64(ps.MaxReceived.Offset)
		ps.Mu.RUnlock()
	}

	s.writeJSON(w, map[string]interface{}{
		"scenario":      scenario,
		"iterations":    iters,
		"duration_s":    duration.Seconds(),
		"ops_per_s":     float64(iters) / duration.Seconds(),
		"state_size":    stateSize,
		"last_commited": lastCommited,
		"max_received":  maxReceived,
	})
}

func (s *BenchmarkServer) writeJSON(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func updateMemStats() *runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryAllocMB.Set(float64(m.Alloc) / 1024 / 1024)
	goroutinesCount.Set(float64(runtime.NumGoroutine()))
	return &m
}

func main() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateMemStats()
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	NewBenchmarkServer(1000)

	// s := NewBenchmarkServer(1000)
	// s.enableProm()
	// http.HandleFunc("/health", s.handleHealth)
	// http.HandleFunc("/scenario/read-heavy", s.handleReadHeavy)
	// http.HandleFunc("/scenario/balanced", s.handleBalanced)
	// http.HandleFunc("/scenario/kafka-sim", s.handleKafkaSim)
	// http.HandleFunc("/reset", s.handleReset)
	// http.Handle("/metrics", promhttp.Handler())

	log.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
