.PHONY: build run

build:
	@go build -o ./bin/app ./...
	@chmod +X ./bin/app

run: build
	@./bin/app


# ------------PARTITION-STATE TESTS-----------
test-lock: build
	@go test -bench=Benchmark_Lock -memprofile=mem-lock.prof

test-chans: build
	@go test -bench=Benchmark_Chans -memprofile=mem-chans.prof

test-sm: build
	@go test -bench=Benchmark_SyncMap -memprofile=mem-sm.prof

combine-results:
	@go test -bench=Benchmark_ParseResults

test-complete: build
	@go test -race -bench=Benchmark_ComprehensiveSeparate
# 	@go test -bench=Benchmark_ComprehensiveComparison -race -benchtime=1000000x -memprofile=mem-smro.prof -benchmem .


# ------------STRINGS TESTS-----------
test-str-concat: 
	@go test -bench=Benchmark_Concat -memprofile=mem-concat.prof

test-str-builder: 
	@go test -bench=Benchmark_Builder -memprofile=mem-builder.prof 

