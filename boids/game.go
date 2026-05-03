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

var drawInt int

type Game struct {
	sg              *SpiralGrid
	boids           []*Boid
	activeBoidCount int // actual count used this game (from ActiveBoidsCount)
	jobsCH          chan (int)
	accels          [boidsCount]Vector2D
	wg              *sync.WaitGroup

	bgImage           *ebiten.Image
	bgCollisionMask   *BgCollisionMask
	barnCollisionMask *BarnCollisionMask

	players   [2]*Player
	aiPlayers [2]*AIPlayer
	winner    *Player
	barns     [2]*Barn

	eventManager *EventManager
	winFN        WinSetter
	hadEnded     bool
}

func NewGame(winFN WinSetter) *Game {
	accels := [boidsCount]Vector2D{}

	// Clamp ActiveBoidsCount to the compile-time upper bound.
	activeBoidCount := ActiveBoidsCount
	if activeBoidCount > boidsCount {
		activeBoidCount = boidsCount
	}
	if activeBoidCount < 1 {
		activeBoidCount = 1
	}

	g := &Game{
		jobsCH:          make(chan int, boidsCount),
		accels:          accels,
		wg:              new(sync.WaitGroup),
		sg:              NewSpiralGrid(screenWidth / 10),
		barns:           [2]*Barn{},
		winFN:           winFN,
		activeBoidCount: activeBoidCount,
	}

	g.loadBarnMask()
	g.barns[0] = NewBarn(true, barnSizeX/2+barnOffsetX, screenHeight/2, g.barnCollisionMask)
	g.barns[1] = NewBarn(false, screenWidth-(barnSizeX/2+barnOffsetX), screenHeight/2, g.barnCollisionMask)

	g.loadBgImg()

	collChecker := g.buildCollisionChecker()

	sheepImg := NewSheepImage(sheepSheet, 5)
	g.eventManager = NewEventManager(collChecker, bananaEventSheet, bananaPeelSheet, energySheet)

	boids := make([]*Boid, activeBoidCount)
	for id := range activeBoidCount {
		b := NewBoid(id, sheepImg, collChecker, g.buildGateChecker())
		boids[id] = b
		g.sg.Insert(b)
	}

	g.boids = boids

	g.players[0] = NewPlayer(collChecker, &PlayerOpts{
		isLeft:       true,
		charaterType: ActiveCharP1,
		IsAI:         ActiveAIPlayer == 1 || ActiveAIPlayer == 3,
	})
	g.players[1] = NewPlayer(collChecker, &PlayerOpts{
		isLeft:       false,
		charaterType: ActiveCharP2,
		IsAI:         ActiveAIPlayer == 2 || ActiveAIPlayer == 3,
	})
	if ActiveAIPlayer == 1 || ActiveAIPlayer == 3 {
		g.aiPlayers[0] = NewAIPlayer(g.players[0], g.barns[0], g.boids, g.eventManager)
	}
	if ActiveAIPlayer == 2 || ActiveAIPlayer == 3 {
		g.aiPlayers[1] = NewAIPlayer(g.players[1], g.barns[1], g.boids, g.eventManager)
	}

	g.StartJobs()
	return g
}

func (g *Game) StartJobs() {
	cpus := runtime.NumCPU()
	for i := range int(cpus / 4) {
		go func(i int) {
			neibBuf := []int{}
			for id := range g.jobsCH {
				b := g.boids[id]
				if b.state == StateFlocking {
					g.sg.GetNeighbours(b, &neibBuf)
				}
				acc := b.calcAcceleration(g, neibBuf, g.players)
				g.accels[id] = acc
				neibBuf = neibBuf[:0]
				g.wg.Done()
			}
		}(i)
	}
}

func (g *Game) Update() error {
	if g.winner != nil || g.hadEnded {
		return nil
	}

	winThreshold := g.activeBoidCount / 3

	for _, b := range g.barns {
		if b.SheepCount >= winThreshold {
			for _, p := range g.players {
				if b.isLeft == p.isLeft {
					g.winFN(p)
					g.winner = p
					return nil
				}
			}
		}
	}

	kept := g.eventManager.drawItems[:0]
	for _, event := range g.eventManager.drawItems {
		if event.IsDone() {
			g.eventManager.ramEventCount--
			continue
		}
		p := findNearestPlayer(g.players, event.GetPosition())
		event.Update(p)
		if event.IsCollidingWith(p) && event.EventType() == BananaPeel {
			p.Slip()
			continue
		}
		kept = append(kept, event)
	}
	g.eventManager.drawItems = kept

	g.sg.Clean()

	for _, p := range g.players {
		if !p.IsAI {
			p.Update()
		}
	}
	for _, p := range g.aiPlayers {
		if p != nil {
			p.Update()
		}
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

	g.wg.Add(g.activeBoidCount)
	for _, b := range g.boids {
		g.jobsCH <- b.id
	}
	g.wg.Wait()

	for _, b := range g.boids {
		acc := g.accels[b.id]
		b.Update(acc)
	}

	UpdateScoreAnim()

	if g.winner != nil {
		g.Close()
	}

	return nil
}

func (g *Game) DrawBG(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bgScaleX, bgScaleY)
	screen.DrawImage(g.bgImage, op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.winner != nil || g.hadEnded {
		return
	}

	g.DrawBG(screen)
	for _, barn := range g.barns {
		barn.Draw(screen)
	}

	g.drawScore(screen)

	for _, boid := range g.boids {
		boid.Draw(screen)
	}
	sharedBatch.Flush(screen, sheepSheet)

	for _, p := range g.players {
		p.Draw(screen)
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

func (g *Game) Close() {
	if g.hadEnded {
		return
	}

	close(g.jobsCH)
	g.hadEnded = true
}

func (g *Game) loadBgImg() {
	g.bgImage = bgImage
	g.bgCollisionMask = NewBgCollisionMask(rawBgImage)
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
	const margin = 24.0

	drawScoreSprite(screen, g.barns[0].SheepCount, margin, margin)

	rightW := measureScoreSprite(g.barns[1].SheepCount)
	drawScoreSpriteRight(screen, g.barns[1].SheepCount, screenWidth-rightW-margin, margin)
}
