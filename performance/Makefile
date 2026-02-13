.PHONY: build run proto

build:
	@go build -o ./bin/app ./...
	@chmod +X ./bin/app

run: build
	@./bin/app

# ---PROTO---
proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/test.proto

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


# ------------PLAYGROUND TESTS-----------
test-str-concat: 
	@go test -bench=Benchmark_Concat -memprofile=mem-concat.prof

test-str-builder: 
	@go test -bench=Benchmark_Builder -memprofile=mem-builder.prof 

test-tp-json: 
	@go test -bench=Benchmark_ThrouputJSON -benchtime=1s

test-tp-proto: 
	@go test -bench=Benchmark_ThrouputPROTO -benchtime=1s

test-tp-lm: 
	@go test -bench=Benchmark_ParallelLockMap -benchtime=1s

test-tp-sm: 
	@go test -bench=Benchmark_ParallelSyncMap -benchtime=1s


