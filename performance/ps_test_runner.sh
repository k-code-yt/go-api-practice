echo "-------STARTING TESTS-------"

go build -o ./bin/app ./...

go test -bench=Benchmark_SyncMap
go test -bench=Benchmark_Chans 
go test -bench=Benchmark_Lock 

sleep 2

echo "-------COMBINING RESULTS-------"
go test -bench=Benchmark_ParseResults

echo "-------DONE-------"