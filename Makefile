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
