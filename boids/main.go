package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenHeight   = 640.00 * 2
	screenWidth    = 1280.00 * 2
	boidsCount     = 300
	boidSize       = 7
	fishTargetSize = 75

	alignRadius = fishTargetSize * 1.5
	alightForce = 0.05

	cohRadius = fishTargetSize * 2
	cohForce  = 0.005

	sepRadius = fishTargetSize / 2
	sepForce  = 1

	minSpeed = 1
	maxSpeed = 4

	wallSepDistance = screenWidth / 15
	wallSepForce    = 0.75
)

var (
	fishImages []*ebiten.Image
)

type Game struct {
	boids []*Boid
}

func NewGame() *Game {
	g := &Game{}

	boids := make([]*Boid, boidsCount)
	for id := range boidsCount {
		randIdx := rand.Intn(len(fishImages))
		randFish := fishImages[randIdx]
		b := NewBoid(id, randFish)
		boids[id] = b
	}

	g.boids = boids
	return g
}

func (g *Game) Update() error {
	for _, b := range g.boids {
		b.Update(g)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// screen.Fill(color.RGBA{255, 245, 228, 0xff})

	for _, b := range g.boids {
		b.Draw(screen)
	}
	fps := fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS())
	ebitenutil.DebugPrint(screen, fps)
}

func (g *Game) Layout(_, _ int) (sw, sh int) {
	return screenWidth, screenHeight
}

func init() {
	dirPath := "./assets"
	dir, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}
	for _, f := range dir {
		f.Info()
		if strings.HasSuffix(f.Name(), ".png") && strings.Contains(f.Name(), "fish") {
			img, _, err := ebitenutil.NewImageFromFile(fmt.Sprintf("%s/%s", dirPath, f.Name()))
			if err != nil {
				panic(err)
			}
			fishImages = append(fishImages, img)
		}
	}
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
