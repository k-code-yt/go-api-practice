package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type Boid struct {
	position        Vector2D
	velocity        Vector2D
	id              int
	img             *SheepImage
	frameIdx        int
	frameTick       int
	facingLeft      bool
	bgCollisionMask *BgCollisionMask
}

func NewBoid(id int, img *SheepImage, bgCollisionMask *BgCollisionMask) *Boid {
	borderMargin := 0.2
	position := Vector2D{rand.Float64() * screenWidth, rand.Float64() * screenHeight}
	velocity := Vector2D{(rand.Float64() * 2) - 1, (rand.Float64() * 2) - 1}

	if position.x < screenWidth*borderMargin {
		position.x = screenWidth * borderMargin
	}
	if position.y < screenHeight*borderMargin {
		position.y = screenHeight * borderMargin
	}
	if position.x > screenWidth*(1-borderMargin) {
		position.x = screenWidth * (1 - borderMargin)
	}
	if position.y > screenHeight*(1-borderMargin) {
		position.y = screenHeight * (1 - borderMargin)
	}

	b := &Boid{
		id:              id,
		velocity:        velocity,
		position:        position,
		img:             img,
		frameIdx:        rand.Intn(img.frameCount),
		bgCollisionMask: bgCollisionMask,
	}
	return b
}

func (b *Boid) Update(accel *Vector2D) {
	b.velocity = b.velocity.Add(*accel).LimitSpeed()
	b.position = b.position.Add(b.velocity)
	b.invertOnWall()
	maxTickPerFrame := 12

	ln := b.velocity.Len()
	tickPerFrame := int(math.Max(minSpeed, float64(maxTickPerFrame)-ln))
	b.frameTick++
	if b.frameTick >= tickPerFrame {
		b.frameTick = 0
		b.frameIdx = (b.frameIdx + 1) % b.img.frameCount

		if b.velocity.x > 0.5 {
			b.facingLeft = false
		} else if b.velocity.x < 0.5 {
			b.facingLeft = true
		}

	}
	// -------
}

func (b *Boid) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	img := b.img
	frame := b.img.Frame(b.frameIdx)

	// TODO -> add angle
	// vx, vy := b.velocity.x, b.velocity.y
	// scaleX := img.scaleX

	// angle := math.Atan2(vy, math.Abs(vx)) - math.Pi/2
	// maxTilt := math.Pi / 18
	// angle = math.Max(-maxTilt, math.Min(maxTilt, angle))

	// op.GeoM.Scale(scaleX, img.scaleY)
	// op.GeoM.Rotate(angle)
	// ---

	scaleX := img.scaleX
	if b.facingLeft {
		scaleX = -scaleX
	}
	op.GeoM.Translate(-b.img.frameW/2, -b.img.frameH/2)
	op.GeoM.Scale(scaleX, img.scaleY)
	op.GeoM.Translate(b.position.x, b.position.y)

	screen.DrawImage(frame, op)
}

func (b *Boid) calcAcceleration(g *Game, neib *[]int) Vector2D {
	avgVelocity := Vector2D{}
	avgPosition := Vector2D{}
	separation := Vector2D{}
	countCoh := 0.0
	countSep := 0.0

	for _, otherIdx := range *neib {
		other := g.boids[otherIdx]
		dist := b.position.Distance(other.position)
		if dist <= sepRadius {
			push := b.position.Sub(other.position).Div((sepRadius - dist) / dist).Normalize().Mul(sepForce)
			separation = separation.Add(push)
			countSep++
		} else if dist < cohRadius {
			avgVelocity = avgVelocity.Add(other.velocity)
			avgPosition = avgPosition.Add(other.position)
			countCoh++

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

	// wallSep := b.wallSeparation()
	// accel = accel.Add(wallSep)

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

func (b *Boid) invertOnWall() {
	hw := targetBoidSize / 2.0 // half-width of scaled sprite
	hh := targetBoidSize / 2.0 // half-height of scaled sprite
	px, py := b.position.x, b.position.y

	// --- Horizontal: check leading X edge ---
	if b.velocity.x > 0 {
		ex := px + hw
		if b.bgCollisionMask.IsBush(ex, py-hh*0.4) ||
			b.bgCollisionMask.IsBush(ex, py) ||
			b.bgCollisionMask.IsBush(ex, py+hh*0.4) {
			b.velocity.x = -b.velocity.x
		}
	} else if b.velocity.x < 0 {
		ex := px - hw
		if b.bgCollisionMask.IsBush(ex, py-hh*0.4) ||
			b.bgCollisionMask.IsBush(ex, py) ||
			b.bgCollisionMask.IsBush(ex, py+hh*0.4) {
			b.velocity.x = -b.velocity.x
		}
	}

	// --- Vertical: check leading Y edge ---
	if b.velocity.y > 0 {
		ey := py + hh
		if b.bgCollisionMask.IsBush(px-hw*0.4, ey) ||
			b.bgCollisionMask.IsBush(px, ey) ||
			b.bgCollisionMask.IsBush(px+hw*0.4, ey) {
			b.velocity.y = -b.velocity.y
		}
	} else if b.velocity.y < 0 {
		ey := py - hh
		if b.bgCollisionMask.IsBush(px-hw*0.4, ey) ||
			b.bgCollisionMask.IsBush(px, ey) ||
			b.bgCollisionMask.IsBush(px+hw*0.4, ey) {
			b.velocity.y = -b.velocity.y
		}
	}
}
