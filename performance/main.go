package main

func main() {
	cfg := NewTestConfig(2000, 1000, 1000, 100, true)

	ps := NewPartitionState(cfg)

	go ps.appendLoop()
	go ps.commitLoop()
	go ps.updateLoop()

	<-ps.ctx.Done()
}
