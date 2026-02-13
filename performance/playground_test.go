package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	testproto "github.com/k-code-yt/go-api-practice/performance/proto"
	"google.golang.org/protobuf/proto"
)

const iters = 15

func runStringsConcat(b *testing.B, numG int, config BenchConfig) {
	fmt.Println(b.N)
	str := "name"
	b.ReportAllocs()
	time.Sleep(time.Second)
	for range iters {
		str += str
	}
}

func runStringsBuilder(b *testing.B, numG int, config BenchConfig) {
	str := []byte("name")
	strByteLen := len(str)
	builder := strings.Builder{}
	time.Sleep(time.Second)
	builder.Grow(iters * strByteLen)

	b.ReportAllocs()
	for i := 0; i < iters; i++ {
		for _, b := range str {
			builder.WriteByte(b)
		}
	}

}

func Benchmark_Concat(b *testing.B) {
	impl := Implementation{
		Name:    "Str-concat",
		RunFunc: runStringsConcat,
	}

	runBenchmark(
		fmt.Sprintf("%s_G%d_%s", impl.Name, GoroutineCounts, impl.Name),
		impl.Name,
		impl.Name,
		GoroutineCounts,
		func() { impl.RunFunc(b, GoroutineCounts, AppedHeavyConfig) },
	)

}

func Benchmark_Builder(b *testing.B) {
	impl := Implementation{
		Name:    "Str-builder",
		RunFunc: runStringsBuilder,
	}
	runBenchmark(
		fmt.Sprintf("%s_G%d_%s", impl.Name, GoroutineCounts, impl.Name),
		impl.Name,
		impl.Name,
		GoroutineCounts,
		func() { impl.RunFunc(b, GoroutineCounts, AppedHeavyConfig) },
	)
}

var result []byte
var totalBytes = 0

func Benchmark_CustomMetric(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data := make([]byte, 1024)
		totalBytes += len(data)
		result = data
	}

	b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op-custom")
	b.ReportMetric(float64(totalBytes)/(1024*1024), "totalMB")
}

var mbVal = 1024

func Benchmark_Throuput(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{mbVal, mbVal * mbVal, 5 * mbVal * mbVal}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size = %dMB", size/mbVal), func(b *testing.B) {
			data := make([]byte, size)
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				sum := 0
				for _, byt := range data {
					sum += int(byt)
				}
				totalBytes += sum
			}
		})
		el := b.Elapsed().Seconds()
		fmt.Printf("test dur = %fs\n", el)
	}
}

type TestSt struct {
	Name           string `json:"Name"`
	Implementation string `json:"Implementation"`
	Goroutines     int    `json:"Goroutines"`
	Scenario       string `json:"Scenario"`
	Data           []byte
}

var jsonRes []byte

func Benchmark_ThrouputJSON(b *testing.B) {
	b.ReportAllocs()

	testJson := TestSt{
		Name:           "john doe",
		Implementation: "test",
		Goroutines:     1,
		Scenario:       "test",
		// Data:           make([]byte, 1024),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(testJson)
		b.SetBytes(int64(len(data)))
		jsonRes = data
	}
}

var protoRes []byte

func Benchmark_ThrouputPROTO(b *testing.B) {
	b.ReportAllocs()

	testProto := testproto.TestSt{
		Name:           "john doe",
		Implementation: "test",
		Goroutines:     1,
		Scenario:       "test",
		// Data:           make([]byte, 1024),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, _ := proto.Marshal(&testProto)
		b.SetBytes(int64(len(data)))
		protoRes = data
	}
}

type LockMap struct {
	mu    *sync.RWMutex
	state map[int]TestSt
}

var testJson = TestSt{
	Name:           "john doe",
	Implementation: "test",
	Goroutines:     1,
	Scenario:       "test",
	// Data:           make([]byte, 1024),
}

var lm = LockMap{
	mu:    new(sync.RWMutex),
	state: make(map[int]TestSt),
}

var maxprocs = 8

func Benchmark_ParallelLockMap(b *testing.B) {
	b.ReportAllocs()
	runtime.GOMAXPROCS(maxprocs)

	b.RunParallel(func(p *testing.PB) {
		i := 0
		for p.Next() {
			lm.mu.Lock()
			lm.state[i] = testJson
			lm.mu.Unlock()
			i++
		}
	})
}

var sm = sync.Map{}

func Benchmark_ParallelSyncMap(b *testing.B) {
	b.ReportAllocs()
	runtime.GOMAXPROCS(maxprocs)

	b.RunParallel(func(p *testing.PB) {
		i := 0
		for p.Next() {
			sm.Store(i, testJson)
			i++
		}
	})
}
