.PHONY: build-producer run clean


build-producer:
	@go build -o ./protocol-playground/bin/data-producer ./protocol-playground/data-producer/...
	@chmod +x ./protocol-playground/bin/data-producer

producer: build-producer
	@./protocol-playground/bin/data-producer

build-receiver:
	@go build -o ./protocol-playground/bin/data-receiver ./protocol-playground/data-receiver/...
	@chmod +x ./protocol-playground/bin/data-receiver

receiver: build-receiver
	@./protocol-playground/bin/data-receiver

build-aggregator:
	@go build -o ./protocol-playground/bin/data-aggregator ./protocol-playground/data-aggregator/
	@chmod +x ./protocol-playground/bin/data-aggregator

aggr: build-aggregator
	@./protocol-playground/bin/data-aggregator

build-invoicer:
	@go build -o ./protocol-playground/bin/invoicer ./protocol-playground/invoicer/
	@chmod +x ./protocol-playground/bin/invoicer

inv: build-invoicer
	@./protocol-playground/bin/invoicer


build-file-sender:
	@go build -o ./protocol-playground/bin/file-sender ./protocol-playground/file-sender/
	@chmod +x ./protocol-playground/bin/file-sender

file: build-file-sender
	@./protocol-playground/bin/file-sender


build-chat:
	@go build -o ./chat/bin/chat ./chat/.
	@chmod +x ./chat/bin/chat

chat: build-chat
	@./chat/bin/chat

test-bp-race:
	@go clean -testcache
	@echo "Running tests with race detector..."
	@go test -race -v -timeout 300s -run TestBackPressure ./chat/with-loop-per-client

test-thr-race:
	@go clean -testcache
	@echo "Running tests with race detector..."
	@go test -race -v -timeout 300s -run TestThrottling ./chat/with-loop-per-client

test-ratelimiter-race:
	@go clean -testcache
	@echo "Running tests with race detector..."
	@go test -race -v -timeout 60s -run TestRequestRateLimitter ./chat/ratelimiter

test-mem-aloc:
	@go clean -testcache
	@echo "Running benchmark test..."
	@go test -v -run TestBench ./tests

test-mem-profile:
	@go clean -testcache
	@echo "Running benchmark test ..."
	@go test -bench=. -benchmem -memprofile=mem.prof ./chat/ratelimiter

test-chat:
	@go clean -testcache
	@echo "Running tests for chat"
	@go test -v ./chat/...

test-chat-race:
	@go clean -testcache
	@echo "Running TestRoomsWithKafka with race detector..."
	@go test -race -v -timeout 300s -run TestRoomsWithKafka ./chat/with-loop-per-client

# ---PROTO---s
proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative shared/ptypes.proto

# ---KAFKA with MUTEX---
build-kafka-app:
	@go build -o ./kafka/bin/kafka ./kafka/cmd/server/.
	@chmod +x ./kafka/bin/kafka

kafka-app: build-kafka-app
	@CG_ID=$(CG_ID) SHOULD_PRODUCE=$(SHOULD_PRODUCE) ./kafka/bin/kafka

build-kafka-app-race:
	@go build -race -o ./kafka/bin/kafka ./kafka/cmd/server/.
	@chmod +x ./kafka/bin/kafka

kafka-app-race: build-kafka-app-race
	@./kafka/bin/kafka

kafka-c-1: build-kafka-app-race
	@CG_ID=consumer-group-1 SHOULD_PRODUCE=false ./kafka/bin/kafka

kafka-c-2: build-kafka-app-race
	@CG_ID=consumer-group-2 SHOULD_PRODUCE=false ./kafka/bin/kafka

kafka-p: build-kafka-app
	@CG_ID=producer-group SHOULD_PRODUCE=true ./kafka/bin/kafka

	

# ---KAFKA with CHANS---
build-kafka-chans-server:
	@go build -o ./kafka-chans/bin/server ./kafka-chans/cmd/server/.
	@chmod +x ./kafka-chans/bin/server

kafka-app-chans: build-kafka-chans-server
	@CG_ID=$(CG_ID) SHOULD_PRODUCE=$(SHOULD_PRODUCE) ./kafka-chans/bin/server/server/.

build-kafka-chans-server-race:
	@go build -race -o ./kafka-chans/bin/server ./kafka-chans/cmd/server/.
	@chmod +x ./kafka-chans/bin/server

kafka-c-chans-1: build-kafka-chans-server-race
	@CG_ID=consumer-group-1 SHOULD_PRODUCE=false ./kafka-chans/bin/server

kafka-c-chans-2: build-kafka-chans-server-race
	@CG_ID=consumer-group-2 SHOULD_PRODUCE=false ./kafka-chans/bin/server

kafka-p-chans: build-kafka-chans-server
	@CG_ID=producer-group SHOULD_PRODUCE=true ./kafka-chans/bin/server


# ---Benchmarking-mem---
build-kafka-chans-benchmark:
	@go build -o ./kafka-chans/bin/benchmark ./kafka-chans/cmd/benchmark/.
	@chmod +x ./kafka-chans/bin/benchmark

build-kafka-mutex-benchmark:
	@go build -o ./kafka/bin/benchmark ./kafka/cmd/benchmark/.
	@chmod +x ./kafka/bin/benchmark

kafka-chans-read-test: build-kafka-chans-benchmark
	@go test -bench=BenchmarkPS_ReadHeavy -benchtime=1000000x -memprofile=mem-chans-read.prof -benchmem ./kafka-chans/cmd/benchmark

kafka-mutex-read-test: build-kafka-mutex-benchmark
	@go test -bench=BenchmarkPS_ReadHeavy -benchtime=1000000x -memprofile=mem-mutex-read.prof -benchmem ./kafka/cmd/benchmark
	
kafka-chans-balanced-test: build-kafka-chans-benchmark
	@go test -bench=BenchmarkPS_Balanced -benchtime=1000000x -memprofile=mem-chans-balanced.prof -benchmem ./kafka-chans/cmd/benchmark

kafka-mutex-balanced-test: build-kafka-mutex-benchmark
	@go test -bench=BenchmarkPS_Balanced -benchtime=1000000x -memprofile=mem-mutex-balanced.prof -benchmem ./kafka/cmd/benchmark

kafka-chans-actual-test: build-kafka-chans-benchmark
	@go test -bench=BenchmarkPS_KafkaSim -benchtime=1000000x -memprofile=mem-chans-kakfa-sim.prof -benchmem ./kafka-chans/cmd/benchmark

kafka-mutex-actual-test: build-kafka-mutex-benchmark
	@go test -bench=BenchmarkPS_KafkaSim -benchtime=1000000x -memprofile=mem-mutex-kakfa-sim.prof -benchmem ./kafka/cmd/benchmark

# ---Benchmarking-cpu---
kafka-chans-test-cpu: build-kafka-chans-benchmark
	@go test -bench=BenchmarkPS_FindLatestToCommit -benchtime=1000000x -cpuprofile=cpu-chans.prof -benchmem ./kafka-chans/cmd/benchmark

kafka-mutex-test-cpu: build-kafka-mutex-benchmark
	@go test -bench=BenchmarkPS_FindLatestToCommit -benchtime=1000000x -cpuprofile=cpu-mutex.prof -benchmem ./kafka/cmd/benchmark

