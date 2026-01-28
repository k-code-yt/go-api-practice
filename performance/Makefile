.PHONY: build

build:
	@go build -o ./bin/main ./...
	@chmod +X ./bin/main

run: build
	@./bin/main

test-uh: build
# 	@go test -bench=Benchmark_UpdateHeavy 
# 	@go test -bench=Benchmark_UpdateHeavy -benchtime=1000000x -memprofile=mem.prof -benchmem .
	@go test -bench=Benchmark_UpdateHeavy -memprofile=mem.prof -benchmem .