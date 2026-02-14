package main

import (
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenHeight = 640
	screenWidth  = 1280
	boidsCount   = 300
	boidSize     = 7
)

var (
	adjRate    = 0.015
	viewRadius = 15.00
)

type Game struct {
	boids    []*Boid
	boidsMap [screenWidth + 1][screenHeight + 1]int
	mapMu    *sync.RWMutex
}

func NewGame() *Game {
	g := &Game{}

	g.mapMu = new(sync.RWMutex)
	for i, row := range g.boidsMap {
		for j := range row {
			g.boidsMap[i][j] = -1
		}
	}

	boids := make([]*Boid, boidsCount)
	for id := range boidsCount {
		b := NewBoid(id)
		g.boidsMap[int(b.position.x)][int(b.position.y)] = b.id
		boids[id] = b
		g.boids = boids
	}

	for _, b := range g.boids {
		go b.start(g)
	}

	g.boids = boids
	return g
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// screen.Fill(color.RGBA{255, 245, 228, 0xff})

	for _, b := range g.boids {
		b.Draw(screen)
	}
}

func (g *Game) Layout(_, _ int) (sw, sh int) {
	return screenWidth, screenHeight
}

func main() {
	game := NewGame()

	ebiten.SetWindowTitle("boids game")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	err := ebiten.RunGame(game)
	if err != nil {
		log.Fatal(err)
	}
}
