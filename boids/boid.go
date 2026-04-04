package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type BoidState int

const (
	StateFlocking BoidState = iota
	StateFleeing
	StateCaught
)

type Boid struct {
	position   Vector2D
	velocity   Vector2D
	id         int
	img        *SheepImage
	frameIdx   int
	frameTick  int
	facingLeft bool
	state      BoidState
	caughtTick int

	bgCollisionMask CollisionMask // was *BgCollisionMask — now the interface
}

func NewBoid(id int, img *SheepImage, bgCollisionMask CollisionMask) *Boid {
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

	return &Boid{
		id:              id,
		velocity:        velocity,
		position:        position,
		img:             img,
		frameIdx:        rand.Intn(img.frameCount),
		bgCollisionMask: bgCollisionMask,
	}
}

func (b *Boid) Update(accel *Vector2D, p *Player) {
	b.updateState(p)

	b.velocity = b.velocity.Add(*accel).LimitSpeed()
	b.invertOnWall()
	b.position = b.position.Add(b.velocity)

	maxTickPerFrame := 12
	ln := b.velocity.Len()
	tickPerFrame := int(math.Max(minSpeed, float64(maxTickPerFrame)-ln))
	b.frameTick++
	if b.frameTick >= tickPerFrame {
		b.frameTick = 0
		b.frameIdx = (b.frameIdx + 1) % b.img.frameCount
		if b.velocity.x > 0.5 {
			b.facingLeft = false
		} else if b.velocity.x < -0.5 {
			b.facingLeft = true
		}
	}
}

func (b *Boid) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	scaleX := b.img.scaleX
	if b.facingLeft {
		scaleX = -scaleX
	}
	op.GeoM.Translate(-b.img.frameW/2, -b.img.frameH/2)
	op.GeoM.Scale(scaleX, b.img.scaleY)
	op.GeoM.Translate(b.position.x, b.position.y)
	screen.DrawImage(b.img.Frame(b.frameIdx), op)
}

func (b *Boid) updateState(p *Player) {
	dist := b.position.Distance(p.position)
	switch b.state {
	case StateFlocking:
		if dist < catchRadius {
			b.state = StateCaught
			b.caughtTick = 0
		} else if dist < fleeRadius {
			b.state = StateFleeing
		}
	case StateCaught:
		b.caughtTick++
		if b.caughtTick > caughtTicks {
			b.state = StateFlocking
		}
	case StateFleeing:
		if dist < catchRadius {
			b.state = StateCaught
			b.caughtTick = 0
		} else if dist > fleeRadius {
			b.state = StateFlocking
		}
	}
}

func (b *Boid) calcAcceleration(g *Game, neib []int, p *Player) Vector2D {
	if b.state == StateCaught || b.state == StateFleeing {
		return b.fleeAccel(p)
	}

	avgVelocity := Vector2D{}
	avgPosition := Vector2D{}
	separation := Vector2D{}
	countCoh := 0.0
	countSep := 0.0

	for _, otherIdx := range neib {
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

	accel := Vector2D{b.bounceOnBorder(b.position.x, screenWidth), b.bounceOnBorder(b.position.y, screenHeight)}
	if countCoh > 0 {
		avgVelocity = avgVelocity.Div(countCoh).Sub(b.velocity)
		avgPosition = avgPosition.Div(countCoh).Sub(b.position)
		accel = accel.Add(avgVelocity.Normalize().Mul(alightForce))
		accel = accel.Add(avgPosition.Normalize().Mul(cohForce))
	}
	if countSep > 0 {
		accel = accel.Add(separation.Div(countSep))
	}

	return accel
}

func (b *Boid) fleeAccel(p *Player) Vector2D {
	return b.position.Sub(p.position).Normalize().Mul(fleeForce)
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

func (b *Boid) wallSeparation() Vector2D {
	wallSep := Vector2D{}
	if b.position.x < wallSepDistance {
		wallSep.x += wallSepForce * (wallSepDistance - b.position.x) / wallSepDistance
	}
	if b.position.x > screenWidth-wallSepDistance {
		wallSep.x -= wallSepForce * (b.position.x - (screenWidth - wallSepDistance)) / wallSepDistance
	}
	if b.position.y < wallSepDistance {
		wallSep.y += wallSepForce * (wallSepDistance - b.position.y) / wallSepDistance
	}
	if b.position.y > screenHeight-wallSepDistance {
		wallSep.y -= wallSepForce * (b.position.y - (screenHeight - wallSepDistance)) / wallSepDistance
	}
	return wallSep
}

func (b *Boid) invertOnWall() {
	hw := targetBoidSize / 2.0
	hh := targetBoidSize / 2.0
	px, py := b.position.x, b.position.y

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
