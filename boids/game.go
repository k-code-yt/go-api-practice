package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenHeight   = 640 * 2.5
	screenWidth    = 1080.00 * 2.5
	boidsCount     = 500
	boidSize       = 7
	fishTargetSize = 60

	alignRadius = fishTargetSize * 1.5
	alightForce = 0.05

	cohRadius = fishTargetSize * 3
	cohForce  = 0.0025

	sepRadius = fishTargetSize / 2
	sepForce  = 1.5

	minSpeed = 1
	maxSpeed = 4

	wallSepDistance = screenWidth / 15
	wallSepForce    = 0.75

	bgPath = "./assets/bg.gif"
)

var (
	fishImages []*FishImage
)

type Game struct {
	sg *SpiralGrid
	// TODO -> should I remove from here? just keep in sg?
	boids  []*Boid
	jobsCH chan (int)
	accels [boidsCount]*Vector2D
	wg     *sync.WaitGroup

	// TODO -> fix gif MEM usage
	bgFrames []*ebiten.Image
	bgDelays []int // delay per frame in 100ths of a second
	bgFrame  int   // current frame index
	bgTick   int   // counts up each Update()
}

func NewGame() *Game {
	accels := [boidsCount]*Vector2D{}

	g := &Game{
		jobsCH: make(chan int, boidsCount),
		accels: accels,
		wg:     new(sync.WaitGroup),
		sg:     NewSpiralGrid(cohRadius),
	}
	bgs, dels := loadGIF(bgPath)

	g.bgDelays = dels
	g.bgFrames = bgs

	boids := make([]*Boid, boidsCount)
	for id := range boidsCount {
		randIdx := rand.Intn(len(fishImages))
		randFish := fishImages[randIdx]
		b := NewBoid(id, randFish)
		boids[id] = b
		g.sg.Insert(b)
	}

	g.boids = boids
	g.StartJobs()
	return g
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
	// fmt.Printf("CPUS = %d\n", cpus)
	for i := range cpus {
		go func(i int) {
			for id := range g.jobsCH {
				b := g.boids[id]
				neib := g.sg.GetNeighbours(b)
				acc := b.calcAcceleration(g, neib)
				// fmt.Printf("%d worker handling acc for boidID = %d\n", i, id)
				g.accels[id] = &acc
				g.wg.Done()
			}
		}(i)
	}
}

func (g *Game) UpdateBG() {
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
}

func (g *Game) Update() error {
	g.UpdateBG()

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
	// TODO -> remove it from DRAW -> do it New
	op := &ebiten.DrawImageOptions{}
	bg := g.bgFrames[g.bgFrame]
	scaleX := screenWidth / float64(bg.Bounds().Dx())
	scaleY := screenHeight / float64(bg.Bounds().Dy())
	op.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(bg, op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.DrawBG(screen)
	for _, b := range g.boids {
		b.Draw(screen)
	}
	fps := fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS())
	r := rand.Int31n(10)
	if r < 1 {
		fmt.Printf("FPS = %s\n", fps)
	}
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
		// TODO -> understand how and fix if possible
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
			calculatedFish := NewFishImage(img)
			fishImages = append(fishImages, calculatedFish)
		}
	}
}
