.PHONY: build run

build:
	@go build -o bin/boids ./...
	@chmod +x bin/boids

run: build
	./bin/boids


# ---TESTS----
test-main: 
	@go test -bench=BenchmarkMain -cpuprofile=cpu.prof

test-main-race: 
	@go test -race -bench=BenchmarkMain -memprofile=mem.prof -cpuprofile=cpu.prof