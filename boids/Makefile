.PHONY: build run

build:
	@go build -o bin/boids ./...
	@chmod +x bin/boids

run: build
	./bin/boids


# ---TESTS----
test-main: 
	@go test -bench=BenchmarkMain -memprofile=./profiling_results/mem.prof -cpuprofile=./profiling_results/cpu.prof 

test-main-race: 
	@go test -race -bench=BenchmarkMain -memprofile=./profiling_results/mem.prof -cpuprofile=./profiling_results/cpu.prof 


test-mp: 
	@go test -bench=BenchmarkMathPow -cpuprofile=cpu_mult.prof -benchtime=40000000x



# ----PPROF----
compare-cpu:
	@go tool pprof -diff_base=./profiling_results/cpu_with_spiral.prof -http=:8080  ./profiling_results/cpu_without_mathpow_1000b.prof

mem:
	@go tool pprof -diff_base=./profiling_results/mem_with_pool.prof  -http=:8080  ./profiling_results/mem.prof 
# 	@go tool pprof -http=:8080 ./profiling_results/mem.prof 
# 	@go tool pprof ./profiling_results/mem_with_pool.prof 