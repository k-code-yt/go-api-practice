.PHONY: build run

build:
	@go build -o bin/boids ./...
	@chmod +x bin/boids

run: build
	./bin/boids


# ---TESTS----
test-main: 
	@go test -bench=BenchmarkMain -memprofile=mem.prof -cpuprofile=cpu.prof

test-main-race: 
	@go test -race -bench=BenchmarkMain -memprofile=mem.prof -cpuprofile=cpu.prof


BINARY_NAME=boids
BUILD_DIR=.
build-win64:
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)_win.exe .