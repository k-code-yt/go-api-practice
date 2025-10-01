.PHONY: build-producer run clean


build-producer:
	@go build -o ./bin/data-producer ./data-producer/...
	@chmod +x ./bin/data-producer

producer: build-producer
	@./bin/data-producer

build-receiver:
	@go build -o ./bin/data-receiver ./data-receiver/...
	@chmod +x ./bin/data-receiver

receiver: build-receiver
	@./bin/data-receiver

build-aggregator:
	@go build -o ./bin/data-aggregator ./data-aggregator/
	@chmod +x ./bin/data-aggregator

aggr: build-aggregator
	@./bin/data-aggregator

build-invoicer:
	@go build -o ./bin/invoicer ./invoicer/
	@chmod +x ./bin/invoicer

inv: build-invoicer
	@./bin/invoicer

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative shared/ptypes.proto
