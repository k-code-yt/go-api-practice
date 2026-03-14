package main

import (
	"log"
)

const metricsAddr = ":2112"

func main() {
	StartMetricsServer(metricsAddr)
	game := NewGame()
	log.Fatal(game.Run())
}
