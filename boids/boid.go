package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type Boid struct {
	position Vector2D
	velocity Vector2D
	id       int
	img      *FishImage
}

func NewBoid(id int, img *FishImage) *Boid {
	position := Vector2D{rand.Float64() * screenWidth, rand.Float64() * screenHeight}
	velocity := Vector2D{(rand.Float64() * 2) - 1, (rand.Float64() * 2) - 1}

	b := &Boid{
		id:       id,
		velocity: velocity,
		position: position,
		img:      img,
	}
	return b
}

func (b *Boid) Update(accel *Vector2D) {
	b.velocity = b.velocity.Add(*accel).LimitSpeed()
	b.position = b.position.Add(b.velocity)
	b.invertOnWall()
}

func (b *Boid) calcAcceleration(g *Game) Vector2D {
	avgVelocity := Vector2D{}
	avgPosition := Vector2D{}
	separation := Vector2D{}
	countCoh := 0.0
	countSep := 0.0

	for _, other := range g.boids {
		dist := b.position.Distance(other.position)
		if dist < cohRadius && dist > sepRadius {
			avgVelocity = avgVelocity.Add(other.velocity)
			avgPosition = avgPosition.Add(other.position)
			countCoh++
		}

		if dist <= sepRadius {
			push := b.position.Sub(other.position).Div((sepRadius - dist) / dist).Normalize().Mul(sepForce)
			separation = separation.Add(push)
			countSep++
		}
	}

	// accel := Vector2D{b.bounceOnBorder(b.position.x, screenWidth), b.bounceOnBorder(b.position.y, screenHeight)}
	accel := Vector2D{}
	if countCoh > 0 {
		avgVelocity = avgVelocity.Div(countCoh).Sub(b.velocity)
		avgPosition = avgPosition.Div(countCoh).Sub(b.position)
		accelAlign := (avgVelocity.Normalize()).Mul(alightForce)
		accelCoh := avgPosition.Normalize().Mul(cohForce)

		accel = accel.Add(accelAlign).Add(accelCoh)
	}

	if countSep > 0 {
		accelSep := separation.Div(countSep)
		accel = accel.Add(accelSep)
	}

	wallSep := b.wallSeparation()
	accel = accel.Add(wallSep)

	return accel
}

func (b *Boid) wallSeparation() Vector2D {
	wallSep := Vector2D{}

	if b.position.x < wallSepDistance {
		force := wallSepForce * (wallSepDistance - b.position.x) / wallSepDistance
		wallSep.x += force
	}

	widthDiff := screenWidth - wallSepDistance
	if b.position.x > widthDiff {
		force := wallSepForce * (b.position.x - widthDiff) / widthDiff
		wallSep.x -= force
	}

	if b.position.y < wallSepDistance {
		force := wallSepForce * (wallSepDistance - b.position.y) / wallSepDistance
		wallSep.y += force
	}

	hightDiff := screenHeight - wallSepDistance
	if b.position.y > hightDiff {
		force := wallSepForce * (b.position.y - hightDiff) / hightDiff
		wallSep.y -= force
	}

	return wallSep
}

func (b *Boid) bounceOnBorder(min, max float64) float64 {
	if min < cohRadius {
		return 2 / min
	}
	if min > max-cohRadius {
		return 2 / (min - max)
	}
	return 0
}

func (b *Boid) Draw(screen *ebiten.Image) {
	img := b.img

	angle := math.Atan2(b.velocity.y, b.velocity.x)
	op := &ebiten.DrawImageOptions{}

	op.GeoM.Translate(-img.w/2, -img.h/2)
	op.GeoM.Scale(img.scaleX, img.scaleY)
	op.GeoM.Rotate(angle + math.Pi)
	op.GeoM.Translate(b.position.x, b.position.y)

	screen.DrawImage(img.img, op)
}

func (b *Boid) invertOnWall() {
	next := b.position.Add(b.velocity)
	if next.x >= screenWidth || next.x < 0 {
		b.velocity = Vector2D{-b.velocity.x, b.velocity.y}
	}
	if next.y >= screenHeight || next.y < 0 {
		b.velocity = Vector2D{b.velocity.x, -b.velocity.y}
	}
}
