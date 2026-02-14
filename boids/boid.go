package main

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	ev "github.com/hajimehoshi/ebiten/v2/vector"
)

type Boid struct {
	position Vector2D
	velocity Vector2D
	id       int
}

func NewBoid(id int) *Boid {
	position := Vector2D{rand.Float64() * screenWidth, rand.Float64() * screenHeight}
	velocity := Vector2D{(rand.Float64() * 2) - 1, (rand.Float64() * 2) - 1}

	b := &Boid{
		id:       id,
		velocity: velocity,
		position: position,
	}
	return b
}

func (b *Boid) start(g *Game) {
	for {
		b.move(g)
		time.Sleep(5 * time.Millisecond)
	}
}

func (b *Boid) calcAcceleration(g *Game) Vector2D {
	upper, lower := b.position.AddVal(viewRadius), b.position.SubVal(viewRadius)
	avgVelocity := Vector2D{}
	avgPosition := Vector2D{}
	separation := Vector2D{}
	count := 0.0

	for i := math.Max(lower.x, 0); i <= math.Min(upper.x, screenWidth); i++ {
		for j := math.Max(lower.y, 0); j <= math.Min(upper.y, screenHeight); j++ {

			g.mapMu.RLock()
			otherBID := g.boidsMap[int(i)][int(j)]
			g.mapMu.RUnlock()

			if otherBID != -1 && otherBID != b.id {
				otherB := g.boids[otherBID]
				dist := otherB.position.Distance(b.position)
				if dist < viewRadius {
					count++
					avgVelocity = avgVelocity.Add(otherB.velocity)
					avgPosition = avgPosition.Add(otherB.position)
					separation = separation.Add(b.position.Sub(otherB.position).Div(dist).Mul(2))
				}
			}
		}
	}

	accel := Vector2D{b.bounceOnBorder(b.position.x, screenWidth), b.bounceOnBorder(b.position.y, screenHeight)}
	if count > 0 {
		avgVelocity = avgVelocity.Div(count)
		avgPosition = avgPosition.Div(count)
		accelAlign := avgVelocity.Sub(b.velocity).Mul(adjRate)
		accelCoh := avgPosition.Sub(b.position).Mul(adjRate)
		accelSep := Vector2D{}
		if !math.IsNaN(separation.x) && !math.IsNaN(separation.y) {
			accelSep = separation.Mul(adjRate)
		}
		accel = accel.Add(accelAlign).Add(accelCoh).Add(accelSep)
	}
	return accel
}

func (b *Boid) bounceOnBorder(min, max float64) float64 {
	if min < viewRadius {
		return 2 / min
	}
	if min > max-viewRadius {
		return 2 / (min - max)
	}
	return 0
}

func (b *Boid) move(g *Game) {
	accel := b.calcAcceleration(g)
	b.velocity = b.velocity.Add(accel).Limit(-1, 1)
	g.mapMu.Lock()
	g.boidsMap[int(b.position.x)][int(b.position.y)] = -1
	b.position = b.position.Add(b.velocity)
	g.boidsMap[int(b.position.x)][int(b.position.y)] = b.id
	g.mapMu.Unlock()
	next := b.position.Add(b.velocity)

	if next.x >= screenWidth || next.x < 0 {
		b.velocity = Vector2D{-b.velocity.x, b.velocity.y}
	}

	if next.y >= screenHeight || next.y < 0 {
		b.velocity = Vector2D{b.velocity.x, -b.velocity.y}
	}
}

func (b *Boid) Draw(screen *ebiten.Image) {
	ev.FillCircle(screen, float32(b.position.x), float32(b.position.y), boidSize, color.RGBA{255, 148, 148, 0xff}, true)
}
