package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const metricsAddr = ":2112"

func main() {
	// StartMetricsServer(metricsAddr)
	sm := NewSceneManager()
	ebiten.SetWindowTitle("boids game")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	log.Fatal(ebiten.RunGame(sm))
}
