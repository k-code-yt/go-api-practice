.PHONY: build run

build:
	@go build -o bin/boids ./...
	@chmod +x bin/boids

run: build
	./bin/boids