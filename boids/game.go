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

	_ "image/jpeg"
	_ "image/png"
)

// TODO -> move to separate file
var (
	sheepSheet       *ebiten.Image
	barnSheet        *ebiten.Image
	bananaEventSheet *ebiten.Image
	bananaPeelSheet  *ebiten.Image
	energySheet      *ebiten.Image
	ramSheet         *ebiten.Image
	sparkleSheet     *ebiten.Image
	dizzySheet       *ebiten.Image
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

	players [2]*Player
	barns   [2]*Barn

	eventManager *EventManager
}

func NewGame() *Game {
	accels := [boidsCount]*Vector2D{}

	g := &Game{
		jobsCH: make(chan int, boidsCount),
		accels: accels,
		wg:     new(sync.WaitGroup),
		sg:     NewSpiralGrid(screenWidth / 4),
		barns:  [2]*Barn{},
	}

	g.loadBarnMask()
	g.barns[0] = NewBarn(true, barnSizeX/2+barnOffsetX, screenHeight/2, g.barnCollisionMask)
	g.barns[1] = NewBarn(false, screenWidth-(barnSizeX/2+barnOffsetX), screenHeight/2, g.barnCollisionMask)

	g.loadBgImg()

	collChecker := g.buildCollisionChecker()
	g.players[0] = NewPlayer(collChecker, &PlayerOpts{isLeft: true, charaterType: KnightCharacter})
	g.players[1] = NewPlayer(collChecker, &PlayerOpts{isLeft: false, charaterType: GirlCharacter})
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
				// TODO -> move to boid
				if b.state == StateFlocking {
					g.sg.GetNeighbours(b, &neibBuf)
				}
				acc := b.calcAcceleration(g, neibBuf, g.players)
				g.accels[id] = &acc
				neibBuf = neibBuf[:0]
				g.wg.Done()
			}
		}(i)
	}
}

func (g *Game) Update() error {
	for i, event := range g.eventManager.drawItems {
		if event.IsDone() {
			g.eventManager.drawItems = append(
				g.eventManager.drawItems[:i],
				g.eventManager.drawItems[i+1:]...,
			)
			g.eventManager.ramEventCount--
			continue
		}
		p := findNearestPlayer(g.players, event.GetPosition())
		event.Update(p)

		if event.IsCollidingWith(p) {
			et := event.EventType()
			switch et {
			case BananaPeel:
				p.Slip()
				g.eventManager.drawItems = append(
					g.eventManager.drawItems[:i],
					g.eventManager.drawItems[i+1:]...,
				)
			default:
			}

		}
	}

	g.sg.Clean()

	for _, p := range g.players {
		p.Update()
	}

	g.eventManager.Update(g.players)

	activeBoids := 0
	for _, b := range g.boids {
		g.sg.Insert(b)
		if b.state != StateInBarn {
			activeBoids++
		}
	}

	g.eventManager.spawnTick++
	if activeBoids%2 == 0 || g.eventManager.spawnTick >= eventSpawnTicks {
		g.eventManager.SpawnPickUp(activeBoids, g.players)
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

func (g *Game) Draw(screen *ebiten.Image) {
	g.DrawBG(screen)
	g.drawScore(screen)

	for _, p := range g.players {
		p.Draw(screen)
	}
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

func (g *Game) drawScore(screen *ebiten.Image) {
	drawScoreSprite(screen, g.barns[0].SheepCount, 24, 24)

	rightW := measureScoreSprite(g.barns[1].SheepCount)
	drawScoreSprite(screen, g.barns[1].SheepCount, screenWidth-rightW-24, 24)
}

// TODO -> refactor -> move to separate files
func init() {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	scoreFont = &text.GoTextFace{
		Source: src,
		Size:   62,
	}

	// ----characters----
	knightImg, _, err := ebitenutil.NewImageFromFile(KnightOpts.sheetPath)
	if err != nil {
		log.Fatal("player sprite:", err)
	}
	KnightOpts.img = knightImg

	girlImg, _, err := ebitenutil.NewImageFromFile(GirlOpts.sheetPath)
	if err != nil {
		log.Fatal("player sprite:", err)
	}
	GirlOpts.img = girlImg

	// ----boids----
	sheep, _, err := ebitenutil.NewImageFromFile(sheepImgPath)
	if err != nil {
		log.Fatal("sheep sprite:", err)
	}
	sheepSheet = sheep

	// ----events----
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

	ram, _, err := ebitenutil.NewImageFromFile(ramPath)
	if err != nil {
		log.Fatal("ram sprite:", err)
	}
	ramSheet = ram

	// ----effects----
	sparkle, _, err := ebitenutil.NewImageFromFile(sparkeSheetPath)
	if err != nil {
		log.Fatal("sparkle sprite:", err)
	}
	sparkleSheet = sparkle

	dizzy, _, err := ebitenutil.NewImageFromFile(dizzySheetPath)
	if err != nil {
		log.Fatal("dizzy sprite:", err)
	}
	dizzySheet = dizzy

	initScoreDisplay()
	loadDizzyFrames()
}
