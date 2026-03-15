package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	_ "image/jpeg"
	_ "image/png"
)

const (
	screenHeight   = 640 * 1.5
	screenWidth    = 1080.00 * 1.5
	boidsCount     = 200
	boidSize       = 7
	targetBoidSize = 65

	alignRadius = targetBoidSize * 1.5
	alightForce = 0.05

	cohRadius = targetBoidSize * 2.5
	cohForce  = 0.0025

	sepRadius = targetBoidSize / 5
	sepForce  = 2

	minSpeed float64 = 1
	maxSpeed float64 = 4

	wallSepDistance = screenWidth / 15
	wallSepForce    = 0.75

	bgPath       = "./assets/bush_border/bush.png"
	sheepImgPath = "./assets/sheep/sheep_run.png"
)

var (
	sheepSheet  *ebiten.Image
	playerSheet *ebiten.Image
)

type Game struct {
	sg     *SpiralGrid
	boids  []*Boid
	jobsCH chan (int)
	accels [boidsCount]*Vector2D
	wg     *sync.WaitGroup

	bgImage         *ebiten.Image
	bgCollisionMask *BgCollisionMask

	player *Player
}

func NewGame() *Game {
	accels := [boidsCount]*Vector2D{}

	g := &Game{
		jobsCH: make(chan int, boidsCount),
		accels: accels,
		wg:     new(sync.WaitGroup),
		sg:     NewSpiralGrid(cohRadius),
	}

	g.loadBgImg()

	g.player = NewPlayer(g.bgCollisionMask)
	sheepImg := NewSheepImage(sheepSheet, 5)
	boids := make([]*Boid, boidsCount)
	for id := range boidsCount {
		b := NewBoid(id, sheepImg, g.bgCollisionMask)
		boids[id] = b
		g.sg.Insert(b)
	}

	g.boids = boids
	g.StartJobs()
	return g
}

func (g *Game) loadBgImg() {
	img, rawImg, err := ebitenutil.NewImageFromFile(bgPath)
	if err != nil {
		log.Fatal(err)
	}
	g.bgImage = img
	g.bgCollisionMask = NewBgCollisionMask(rawImg)
}

func (g *Game) Run() error {
	ebiten.SetWindowTitle("boids game")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	err := ebiten.RunGame(g)
	if err != nil {
		return err
	}
	return nil
}

func (g *Game) StartJobs() {
	cpus := runtime.NumCPU()
	for i := range int(cpus / 4) {
		go func(i int) {
			neibBuf := []int{}
			for id := range g.jobsCH {
				b := g.boids[id]
				g.sg.GetNeighbours(b, &neibBuf)
				acc := b.calcAcceleration(g, &neibBuf)
				g.accels[id] = &acc
				neibBuf = neibBuf[:0]
				g.wg.Done()
			}
		}(i)
	}
}

func (g *Game) Update() error {
	g.player.Update()
	g.sg.Clean()
	for _, b := range g.boids {
		g.sg.Insert(b)
	}

	g.wg.Add(boidsCount)
	for _, b := range g.boids {
		g.jobsCH <- b.id
	}
	g.wg.Wait()

	for _, b := range g.boids {
		acc := g.accels[b.id]
		b.Update(acc)
	}
	return nil
}

func (g *Game) DrawBG(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	scaleX := screenWidth / float64(g.bgImage.Bounds().Dx())
	scaleY := screenHeight / float64(g.bgImage.Bounds().Dy())
	op.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(g.bgImage, op)
}

var drawInt int

func (g *Game) Draw(screen *ebiten.Image) {
	g.DrawBG(screen)
	g.player.Draw(screen)
	for _, b := range g.boids {
		b.Draw(screen)
	}
	fps := fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS())
	drawInt++
	if drawInt%120 == 0 {
		fmt.Printf("FPS = %s\n", fps)
	}
	ebitenutil.DebugPrint(screen, fps)
}

func (g *Game) Layout(_, _ int) (sw, sh int) {
	return screenWidth, screenHeight
}

func init() {
	sheet, _, err := ebitenutil.NewImageFromFile(sheepImgPath)
	if err != nil {
		log.Fatal(err)
	}
	sheepSheet = sheet

	img, _, err := ebitenutil.NewImageFromFile(playerSheetPath)
	if err != nil {
		log.Fatal("player sprite:", err)
	}
	playerSheet = img

}
