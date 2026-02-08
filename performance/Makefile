.PHONY: build run

build:
	@go build -o ./bin/app ./...
	@chmod +X ./bin/app

run: build
	@./bin/app

test-lock: build
	@go test -bench=Benchmark_Lock 

test-chans: build
	@go test -bench=Benchmark_Chans 

test-sm: build
	@go test -bench=Benchmark_SyncMap

combine-results:
	@go test -bench=Benchmark_ParseResults


test-complete: build
	@go test -race -bench=Benchmark_ComprehensiveSeparate
# 	@go test -bench=Benchmark_ComprehensiveComparison -race -benchtime=1000000x -memprofile=mem-smro.prof -benchmem .


# ------------STRINGS TESTS-----------
test-str-concat: 
	@go test -bench=Benchmark_Concat -benchtime=100x

test-str-builder: 
	@go test -bench=Benchmark_Builder -benchtime=100x

