.PHONY: build run

build:
	@go build -o ./bin/app ./...
	@chmod +X ./bin/app

run: build
	@./bin/app

test-uh: build
# 	@go test -bench=Benchmark_UpdateHeavy 
# 	@go test -bench=Benchmark_UpdateHeavy -benchtime=1000000x -memprofile=mem.prof -benchmem .
	@go test -bench=Benchmark_UpdateHeavy -memprofile=mem.prof -benchmem .

test-lock: build
	@go test -bench=Benchmark_WriteWithLock -benchtime=10000x -memprofile=mem.prof -benchmem .

test-sm: build
	@go test -bench=Benchmark_WriteWithSyncMap -benchtime=10000x -memprofile=mem.prof -benchmem .