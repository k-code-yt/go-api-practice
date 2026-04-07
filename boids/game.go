package main

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"image/color"
	_ "image/jpeg"
	_ "image/png"
)

// TODO -> move to separate file
var (
	sheepSheet       *ebiten.Image
	playerSheet      *ebiten.Image
	barnSheet        *ebiten.Image
	bananaEventSheet *ebiten.Image
	bananaPeelSheet  *ebiten.Image
	energySheet      *ebiten.Image
)
var scoreFont *text.GoTextFace
var drawInt int

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

	eventManager *EventManager
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
	g.player = NewPlayer(collChecker, false)
	sheepImg := NewSheepImage(sheepSheet, 5)
	g.eventManager = NewEventManager(collChecker, bananaEventSheet, bananaPeelSheet, energySheet)

	boids := make([]*Boid, boidsCount)
	for id := range boidsCount {
		b := NewBoid(id, sheepImg, collChecker, g.buildGateChecker())
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
	for i, peel := range g.eventManager.drawItems {
		if peel.IsCollidingWith(g.player) {
			g.player.Slip()
			// TODO(perf) -> optimize slice? ringbuffer?
			// remove this peel from the slice
			g.eventManager.drawItems = append(
				g.eventManager.drawItems[:i],
				g.eventManager.drawItems[i+1:]...,
			)
			break
		}
	}

	g.player.Update()
	g.sg.Clean()
	g.eventManager.HandlePickUpCollision(g.player)

	activeBoids := 0
	for _, b := range g.boids {
		g.sg.Insert(b)
		if b.state != StateInBarn {
			activeBoids++
		}
	}

	if activeBoids%2 == 0 {
		// TODO -> also spawn based on timeout or sheep in barn
		g.eventManager.SpawnPickUp(activeBoids)
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

func (g *Game) Draw(screen *ebiten.Image) {
	g.DrawBG(screen)
	g.drawScores(screen)

	g.player.Draw(screen)
	for _, barn := range g.barns {
		barn.Draw(screen)
	}
	for _, boid := range g.boids {
		boid.Draw(screen)
	}
	if g.eventManager.nextEvent != nil {
		g.eventManager.nextEvent.DrawPickUp(screen)
	}

	g.eventManager.DrawTrigger(screen)

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

func (g *Game) drawScores(screen *ebiten.Image) {
	leftStr := fmt.Sprintf("score:%d", g.barns[0].SheepCount)
	rightStr := fmt.Sprintf("score:%d", g.barns[1].SheepCount)

	drawScore(screen, leftStr, 24, 24,
		color.RGBA{R: 220, G: 50, B: 50, A: 255})

	rightW, _ := text.Measure(rightStr, scoreFont, 0)
	drawScore(screen, rightStr, screenWidth-rightW-24, 24,
		color.RGBA{R: 50, G: 100, B: 220, A: 255})
}

func drawScore(screen *ebiten.Image, str string, x, y float64, col color.RGBA) {
	shadowOp := &text.DrawOptions{}
	shadowOp.GeoM.Translate(x+3, y+3)
	shadowOp.ColorScale.ScaleWithColor(color.RGBA{A: 180})
	text.Draw(screen, str, scoreFont, shadowOp)

	mainOp := &text.DrawOptions{}
	mainOp.GeoM.Translate(x, y)
	mainOp.ColorScale.ScaleWithColor(col)
	text.Draw(screen, str, scoreFont, mainOp)
}

func init() {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	scoreFont = &text.GoTextFace{
		Source: src,
		Size:   62,
	}
	sheep, _, err := ebitenutil.NewImageFromFile(sheepImgPath)
	if err != nil {
		log.Fatal("sheep sprite:", err)
	}
	sheepSheet = sheep

	bananaEvent, _, err := ebitenutil.NewImageFromFile(bananaEventPath)
	if err != nil {
		log.Fatal("bananaEvent sprite:", err)
	}
	bananaEventSheet = bananaEvent
	bananaPeel, _, err := ebitenutil.NewImageFromFile(bananaPeelPath)
	if err != nil {
		log.Fatal("bananaPeel sprite:", err)
	}
	bananaPeelSheet = bananaPeel
	energy, _, err := ebitenutil.NewImageFromFile(energyEventPath)
	if err != nil {
		log.Fatal("energy sprite:", err)
	}
	energySheet = energy

	pImg, _, err := ebitenutil.NewImageFromFile(playerSheetPath)
	if err != nil {
		log.Fatal("player sprite:", err)
	}
	playerSheet = pImg
}
