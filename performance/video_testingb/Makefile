run-test:
	go test -bench=Benchmark_Slice

run-test-loop:
	go test -bench=Benchmark_Loop  -benchtime=6000x

run-test-bn:
	go test -bench=Benchmark_Reset -benchtime=6000x

# ---STRINGS TEST----
run-test-str-concat:
	go test -bench=BenchmarkStrConcat -benchtime=10000x

run-test-str-bdr-ng:
	go test -bench=BenchmarkStrBuilderNoGrow -benchtime=10000x

run-test-str-bdr-grow:
	go test -bench=BenchmarkStrBuilderGrow -benchtime=10000x

