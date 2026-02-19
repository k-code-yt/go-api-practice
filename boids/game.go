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
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	isDebug   = true
	debugTick = time.Second

	screenHeight   = 640 * 2
	screenWidth    = 1080.00 * 2
	boidsCount     = 1000
	boidSize       = 7
	fishTargetSize = 50

	baseRadius    = fishTargetSize * 1.2
	alignRadius   = baseRadius * 1.5
	alignRadiusSq = alignRadius * alignRadius
	alightForce   = 0.05
	alightForceSq = alightForce * alightForce

	cohRadius   = baseRadius * 3
	cohRadiusSq = cohRadius * cohRadius
	cohForce    = 0.0025
	cohForceSq  = cohForce * cohForce

	sepRadius   = baseRadius / 2
	sepRadiusSq = sepRadius * sepRadius
	sepForce    = 1.5
	sepForceSq  = sepForce * sepForce

	minSpeed = 1
	maxSpeed = 4

	wallSepDistance = screenWidth / 15
	wallSepForce    = 0.75

	bgPath = "./assets/bg.gif"
)

var (
	fishImages []*FishImage
)

type Worker struct {
	id      int
	neibBuf []int
}

func NewWorker(id int) *Worker {
	return &Worker{
		id:      id,
		neibBuf: []int{},
	}
}

func (w *Worker) StartJob(g *Game) {
	for id := range g.jobsCH {
		b := g.boids[id]
		g.sg.GetNeighbours(b, &w.neibBuf)
		acc := b.calcAcceleration(g, w.neibBuf)
		g.accels[id] = &acc
		w.neibBuf = w.neibBuf[:0]
		g.wg.Done()
	}

}

type Game struct {
	sg      *SpiralGrid
	boids   []*Boid
	jobsCH  chan (int)
	accels  [boidsCount]*Vector2D
	wg      *sync.WaitGroup
	workers []*Worker

	// TODO -> fix gif MEM usage
	bgFrames []*ebiten.Image
	bgDelays []int // delay per frame in 100ths of a second
	bgFrame  int   // current frame index
	bgTick   int   // counts up each Update()

	// for debug
	ticker time.Ticker
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
	g.StartJob()

	if isDebug {
		g.ticker = *time.NewTicker(debugTick)
	}

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

func (g *Game) StartJob() {
	defer g.ticker.Stop()

	cpus := runtime.NumCPU()

	for i := range cpus {
		w := NewWorker(i)
		go w.StartJob(g)
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
	if isDebug {
		select {
		case <-g.ticker.C:
			fmt.Printf("FPS = %s\n", fps)
		default:
		}
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
