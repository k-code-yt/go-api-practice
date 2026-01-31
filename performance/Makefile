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
	@go test -bench=Benchmark_Lock -benchtime=1000000x -memprofile=mem-lock.prof -benchmem .

test-sm: build
	@go test -bench=Benchmark_SyncMap -benchtime=1000000x -memprofile=mem-sm.prof -benchmem .