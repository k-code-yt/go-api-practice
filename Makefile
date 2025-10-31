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

test-chat:
	@go clean -testcache
	@echo "Running tests for chat"
	@go test -v ./chat/...

test-chat-race:
	@go clean -testcache
	@echo "Running tests with race detector..."
	@go test -race -v ./chat/...

test-bp-race:
	@go clean -testcache
	@echo "Running tests with race detector..."
	@go test -race -v -timeout 300s -run TestBackPressure ./chat/with-loop-per-client


proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative shared/ptypes.proto
