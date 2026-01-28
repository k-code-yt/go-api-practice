.PHONY: build

build:
	@go build -o ./bin/main ./...
	@chmod +X ./bin/main

run: build
	@./bin/main