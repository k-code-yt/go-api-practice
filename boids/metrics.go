package main

import (
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MemSnapshot struct {
	Alloc         uint64    `json:"Alloc"`
	TotalAlloc    uint64    `json:"TotalAlloc"`
	Sys           uint64    `json:"Sys"`
	HeapObjects   uint64    `json:"HeapObjects"`
	HeapInuse     uint64    `json:"HeapInuse"`
	HeapIdle      uint64    `json:"HeapIdle"`
	StackInuse    uint64    `json:"StackInuse"`
	Mallocs       uint64    `json:"Mallocs"`
	Frees         uint64    `json:"Frees"`
	NumGC         uint32    `json:"NumGC"`
	GCCPUFraction float64   `json:"GCCPUFraction"`
	Timestamp     time.Time `json:"Timestamp"`
}

type MemDelta struct {
	AllocDelta         int64         `json:"AllocDelta"`
	TotalAllocDelta    int64         `json:"TotalAllocDelta"`
	SysDelta           int64         `json:"SysDelta"`
	HeapObjectsDelta   int64         `json:"HeapObjectsDelta"`
	HeapInuseDelta     int64         `json:"HeapInuseDelta"`
	HeapIdleDelta      int64         `json:"HeapIdleDelta"`
	StackInuseDelta    int64         `json:"StackInuseDelta"`
	MallocsDelta       int64         `json:"MallocsDelta"`
	FreesDelta         int64         `json:"FreesDelta"`
	NumGCDelta         int32         `json:"NumGCDelta"`
	GCCPUFractionDelta float64       `json:"GCCPUFractionDelta"`
	Duration           time.Duration `json:"Duration"`
}

var (
	memAlloc = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mem_alloc_bytes",
		Help: "Currently allocated heap memory in bytes",
	})
	memTotalAlloc = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mem_total_alloc_bytes_total",
		Help: "Total bytes allocated over lifetime",
	})
	memSys = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mem_sys_bytes",
		Help: "Total memory obtained from OS",
	})
	memHeapObjects = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mem_heap_objects",
		Help: "Number of live heap objects",
	})
	memHeapInuse = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mem_heap_inuse_bytes",
		Help: "Heap memory in use",
	})
	memHeapIdle = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mem_heap_idle_bytes",
		Help: "Heap memory idle",
	})
	memStackInuse = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mem_stack_inuse_bytes",
		Help: "Stack memory in use",
	})
	memMallocs = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mem_mallocs_total",
		Help: "Cumulative mallocs",
	})
	memFrees = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mem_frees_total",
		Help: "Cumulative frees",
	})
	memNumGC = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mem_gc_total",
		Help: "Number of GC cycles completed",
	})
	memGCCPUFraction = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mem_gc_cpu_fraction",
		Help: "Fraction of CPU used by GC",
	})

	// track previous counter values so we only add deltas
	prevTotalAlloc uint64
	prevMallocs    uint64
	prevFrees      uint64
	prevNumGC      uint32
)

func collectMemSnapshot() MemSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	snap := MemSnapshot{
		Alloc:         ms.Alloc,
		TotalAlloc:    ms.TotalAlloc,
		Sys:           ms.Sys,
		HeapObjects:   ms.HeapObjects,
		HeapInuse:     ms.HeapInuse,
		HeapIdle:      ms.HeapIdle,
		StackInuse:    ms.StackInuse,
		Mallocs:       ms.Mallocs,
		Frees:         ms.Frees,
		NumGC:         ms.NumGC,
		GCCPUFraction: ms.GCCPUFraction,
		Timestamp:     time.Now(),
	}

	memAlloc.Set(float64(snap.Alloc))
	memSys.Set(float64(snap.Sys))
	memHeapObjects.Set(float64(snap.HeapObjects))
	memHeapInuse.Set(float64(snap.HeapInuse))
	memHeapIdle.Set(float64(snap.HeapIdle))
	memStackInuse.Set(float64(snap.StackInuse))
	memGCCPUFraction.Set(snap.GCCPUFraction)

	if snap.TotalAlloc > prevTotalAlloc {
		memTotalAlloc.Add(float64(snap.TotalAlloc - prevTotalAlloc))
		prevTotalAlloc = snap.TotalAlloc
	}
	if snap.Mallocs > prevMallocs {
		memMallocs.Add(float64(snap.Mallocs - prevMallocs))
		prevMallocs = snap.Mallocs
	}
	if snap.Frees > prevFrees {
		memFrees.Add(float64(snap.Frees - prevFrees))
		prevFrees = snap.Frees
	}
	if snap.NumGC > prevNumGC {
		memNumGC.Add(float64(snap.NumGC - prevNumGC))
		prevNumGC = snap.NumGC
	}

	return snap
}

func StartMetricsServer(addr string) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			collectMemSnapshot()
		}
	}()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(addr, mux); err != nil {
			panic(err)
		}
	}()
}
