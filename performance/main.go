package main

func main() {
	cfg := NewTestConfig(2000, 1000, 1000, 100, true)

	ps := NewPartitionState(cfg)
	ps.init()
	<-ps.ctx.Done()
}
