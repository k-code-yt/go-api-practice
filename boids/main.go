package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenHeight   = 640 * 2
	screenWidth    = 1080.00 * 2
	boidsCount     = 600
	boidSize       = 7
	fishTargetSize = 50

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

	bgPath = "./assets/bg.gif"
)

var (
	fishImages []*ebiten.Image
)

type Game struct {
	boids    []*Boid
	bgFrames []*ebiten.Image
	bgDelays []int // delay per frame in 100ths of a second
	bgFrame  int   // current frame index
	bgTick   int   // counts up each Update()
}

func NewGame() *Game {
	g := &Game{}
	bgs, dels := loadGIF(bgPath)

	g.bgDelays = dels
	g.bgFrames = bgs

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
	g.bgTick++

	throshold := g.bgDelays[g.bgFrame] * 60 / 100
	if throshold < 1 {
		throshold = 1
	}

	if g.bgTick >= throshold {
		g.bgTick = 0
		lbg := len(g.bgFrames)
		g.bgFrame = (g.bgFrame + 1) % lbg
	}

	for _, b := range g.boids {
		b.Update(g)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	op := &ebiten.DrawImageOptions{}
	bg := g.bgFrames[g.bgFrame]
	scaleX := screenWidth / float64(bg.Bounds().Dx())
	scaleY := screenHeight / float64(bg.Bounds().Dy())
	op.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(bg, op)

	for _, b := range g.boids {
		b.Draw(screen)
	}
	fps := fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS())
	ebitenutil.DebugPrint(screen, fps)
}

func (g *Game) Layout(_, _ int) (sw, sh int) {
	return screenWidth, screenHeight
}

func loadGIF(path string) ([]*ebiten.Image, []int) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		log.Fatal(err)
	}

	frames := make([]*ebiten.Image, len(g.Image))
	delays := make([]int, len(g.Delay))

	bounds := g.Image[0].Bounds()
	rgba := image.NewRGBA(bounds)

	for i, frame := range g.Image {
		draw.Draw(rgba, bounds, frame, bounds.Min, draw.Over)
		frames[i] = ebiten.NewImageFromImage(rgba)
		delays[i] = g.Delay[i]
	}

	return frames, delays
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
