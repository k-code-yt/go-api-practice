.PHONY: build run

build:
	@go build -o ./bin/app ./...
	@chmod +X ./bin/app

run: build
	@./bin/app

test-uh: build
	@go test -bench=Benchmark_All -benchtime=1000000x
# 	@go test -bench=Benchmark_UpdateHeavy -benchtime=1000000x -memprofile=mem.prof -benchmem .
# 	@go test -bench=Benchmark_UpdateHeavy -memprofile=mem.prof -benchmem .

test-lock: build
	@go test -bench=Benchmark_Lock_PS -benchtime=10000000x -memprofile=mem-lock.prof -benchmem .

test-sm: build
	@go test -bench=Benchmark_SyncMap_PS -benchtime=10000000x -memprofile=mem-sm.prof -benchmem .

test-readonly-concy: build
	@go test -bench=Benchmark_ReadOnlyConcurrency -benchtime=10000000x benchmem > readonly_concy.txt

test-mixed-concy: build
	@go test -bench=Benchmark_MixedOnlyConcurrency -benchtime=10000000x benchmem > mixed_concy.txt

test-lockro: build
	@go test -bench=Benchmark_MapReadOnly -benchtime=1000000x -memprofile=mem-lockro.prof -benchmem .

test-smro: build
	@go test -bench=Benchmark_SyncMapReadOnly -benchtime=1000000x -memprofile=mem-smro.prof -benchmem .

test-complete: build
	@go test -race -bench=Benchmark_ComprehensiveSeparate
# 	@go test -bench=Benchmark_ComprehensiveComparison -race -benchtime=1000000x -memprofile=mem-smro.prof -benchmem .