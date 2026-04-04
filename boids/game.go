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

var (
	sheepSheet  *ebiten.Image
	playerSheet *ebiten.Image
	barnSheet   *ebiten.Image
)

type Game struct {
	sg     *SpiralGrid
	boids  []*Boid
	jobsCH chan (int)
	accels [boidsCount]*Vector2D
	wg     *sync.WaitGroup

	bgImage           *ebiten.Image
	bgCollisionMask   *BgCollisionMask
	barnCollisionMask *BarnCollisionMask

	player *Player
	barns  [2]*Barn
}

func NewGame() *Game {
	accels := [boidsCount]*Vector2D{}

	g := &Game{
		jobsCH: make(chan int, boidsCount),
		accels: accels,
		wg:     new(sync.WaitGroup),
		sg:     NewSpiralGrid(cohRadius),
		barns:  [2]*Barn{},
	}

	g.loadBarnMask()
	g.barns[0] = NewBarn(true, barnSizeX/2+barnOffsetX, screenHeight/2, g.barnCollisionMask)
	g.barns[1] = NewBarn(false, screenWidth-(barnSizeX/2+barnOffsetX), screenHeight/2, g.barnCollisionMask)

	g.loadBgImg()

	collChecker := g.buildCollisionChecker()
	g.player = NewPlayer(collChecker)
	sheepImg := NewSheepImage(sheepSheet, 5)

	// gateChecker := g.buildGateChecker()

	boids := make([]*Boid, boidsCount)
	for id := range boidsCount {
		b := NewBoid(id, sheepImg, collChecker)
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
	for i := range int(cpus / 4) {
		go func(i int) {
			neibBuf := []int{}
			for id := range g.jobsCH {
				b := g.boids[id]
				g.sg.GetNeighbours(b, &neibBuf)
				acc := b.calcAcceleration(g, neibBuf, g.player)
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
		b.Update(acc, g.player)
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
	for _, barn := range g.barns {
		barn.Draw(screen)
	}
	for _, boid := range g.boids {
		boid.Draw(screen)
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

func (g *Game) loadBgImg() {
	img, rawImg, err := ebitenutil.NewImageFromFile(bgPath)
	if err != nil {
		log.Fatal(err)
	}
	g.bgImage = img
	g.bgCollisionMask = NewBgCollisionMask(rawImg)
}

func (g *Game) loadBarnMask() {
	barnImg, raw, err := ebitenutil.NewImageFromFile(barnSheetPath)
	if err != nil {
		log.Fatal("barn sprite:", err)
	}
	barnSheet = barnImg

	g.barnCollisionMask = NewBarnCollisionMask(raw)
}

func (g *Game) buildCollisionChecker() CollisionChecker {
	return func(x, y float64) bool {
		if g.bgCollisionMask.IsBush(x, y) {
			return true
		}
		for _, b := range g.barns {
			if b.IsBlocking(x, y) {
				return true
			}
		}
		return false
	}
}

func (g *Game) buildGateChecker() func(x, y float64) *Barn {
	return func(x, y float64) *Barn {
		for _, b := range g.barns {
			if b.IsGate(x, y) {
				return b
			}
		}
		return nil
	}
}

func init() {
	sheep, _, err := ebitenutil.NewImageFromFile(sheepImgPath)
	if err != nil {
		log.Fatal("sheep sprite:", err)
	}
	sheepSheet = sheep

	pImg, _, err := ebitenutil.NewImageFromFile(playerSheetPath)
	if err != nil {
		log.Fatal("player sprite:", err)
	}
	playerSheet = pImg
}
