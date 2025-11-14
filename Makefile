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


build-kafka:
	@go build -o ./kafka/bin/kafka ./kafka/.
	@chmod +x ./kafka/bin/kafka

kafka: build-kafka
	@./kafka/bin/kafka

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative shared/ptypes.proto
