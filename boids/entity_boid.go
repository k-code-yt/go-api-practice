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
	StateMovingToDoor
	StateInBarn
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

	targetX    float64
	targetY    float64
	targetBarn *Barn

	gateChecker GateChecker
	collChecker CollisionChecker

	nearestPlayer *Player
}

func NewBoid(id int, img *SheepImage, collChecker CollisionChecker, gateChecker GateChecker) *Boid {
	position := safeSpawnPosition(collChecker, nil)
	velocity := Vector2D{(rand.Float64() * 2) - 1, (rand.Float64() * 2) - 1}

	b := &Boid{
		id:          id,
		velocity:    velocity,
		position:    position,
		img:         img,
		frameIdx:    rand.Intn(img.frameCount),
		gateChecker: gateChecker,
		collChecker: collChecker,
	}
	return b
}

func (b *Boid) Update(accel Vector2D) {
	b.updateState(b.nearestPlayer)
	b.velocity = b.velocity.Add(accel).LimitSpeed()

	if b.state != StateMovingToDoor && b.state != StateInBarn {
		b.invertOnWall()
	}

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
		} else if b.velocity.x < 0.5 {
			b.facingLeft = true
		}

	}
}

func (b *Boid) Draw(screen *ebiten.Image) {
	if b.state == StateInBarn {
		return
	}
	img := b.img
	frame := b.img.Frame(b.frameIdx)
	bounds := frame.Bounds()

	scaleX := img.scaleX
	if b.facingLeft {
		scaleX = -scaleX
	}
	sharedBatch.Add(
		b.position.x,
		b.position.y,
		bounds.Min.X,
		bounds.Min.Y,
		bounds.Dx(),
		bounds.Dy(),
		int(b.img.w),
		int(b.img.h),
		scaleX,
		b.img.scaleY,
	)
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
		if barn := b.gateChecker(b.position.x, b.position.y); barn != nil {
			b.state = StateMovingToDoor
			b.targetBarn = barn
			b.targetX = barn.x
			b.targetY = barn.GateCenterY()
			b.caughtTick = 0
		} else if b.caughtTick > caughtTicks {
			b.state = StateFlocking
		}
	case StateFleeing:
		if barn := b.gateChecker(b.position.x, b.position.y); barn != nil {
			b.state = StateMovingToDoor
			b.targetBarn = barn
			b.targetX = barn.x
			b.targetY = barn.GateCenterY()
			b.caughtTick = 0
		} else if dist < catchRadius {
			b.state = StateCaught
			b.caughtTick = 0
		} else if dist > fleeRadius {
			b.state = StateFlocking
		}
	case StateMovingToDoor:
		dyLeft := math.Abs(b.position.y - b.targetY)
		if dyLeft > barnEntryTolerance {
			break
		}
		dxLeft := math.Abs(b.position.x - b.targetX)
		if dxLeft < barnEntryTolerance {
			b.state = StateInBarn
			b.targetBarn.SheepCount++
		}
	}

}

func (b *Boid) calcAcceleration(g *Game, neib []int, players [2]*Player) Vector2D {
	p := findNearestPlayer(players, b.position)
	if b.nearestPlayer == nil || b.nearestPlayer != p {
		b.nearestPlayer = p
	}
	switch b.state {
	case StateCaught, StateFleeing:
		return b.fleeAccel(p)
	case StateMovingToDoor:
		return b.moveToDoorAccel()
	case StateInBarn:
		return Vector2D{}
	}

	// Flocking state = boid logic
	avgVelocity := Vector2D{}
	avgPosition := Vector2D{}
	separation := Vector2D{}
	countCoh := 0.0
	countSep := 0.0

	for _, otherIdx := range neib {
		other := g.boids[otherIdx]
		dist := b.position.DistanceSq(other.position)
		if dist <= sepRadius*sepRadius {
			push := b.position.Sub(other.position).Div((sepRadius*sepRadius - dist) / dist).Normalize().Mul(sepForce)
			separation = separation.Add(push)
			countSep++
		} else if dist < cohRadius*cohRadius {
			avgVelocity = avgVelocity.Add(other.velocity)
			avgPosition = avgPosition.Add(other.position)
			countCoh++

		}
	}

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

	return accel
}

func (b *Boid) fleeAccel(p *Player) Vector2D {
	return b.position.Sub(p.position).Normalize().Mul(fleeForce)
}

func (b *Boid) moveToDoorAccel() Vector2D {
	dyLeft := math.Abs(b.position.y - b.targetY)
	if dyLeft > barnEntryTolerance {
		target := Vector2D{b.position.x, b.targetY}
		return target.Sub(b.position).Normalize().Mul(barnEntryForce)
	}
	b.position.y = b.targetY
	target := Vector2D{b.targetX, b.targetY}
	return target.Sub(b.position).Normalize().Mul(barnEntryForce)
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
	hw := targetBoidSize / 2.0
	hh := targetBoidSize / 2.0
	px, py := b.position.x, b.position.y

	// --- Horizontal: check leading X edge ---
	if b.velocity.x > 0 {
		ex := px + hw
		if b.collChecker(ex, py-hh*0.4) ||
			b.collChecker(ex, py) ||
			b.collChecker(ex, py+hh*0.4) {
			b.velocity.x = -b.velocity.x

		}
	} else if b.velocity.x < 0 {
		ex := px - hw
		if b.collChecker(ex, py-hh*0.4) ||
			b.collChecker(ex, py) ||
			b.collChecker(ex, py+hh*0.4) {
			b.velocity.x = -b.velocity.x
		}
	}

	// --- Vertical: check leading Y edge ---
	if b.velocity.y > 0 {
		ey := py + hh
		if b.collChecker(px-hw*0.4, ey) ||
			b.collChecker(px, ey) ||
			b.collChecker(px+hw*0.4, ey) {
			b.velocity.y = -b.velocity.y
		}
	} else if b.velocity.y < 0 {
		ey := py - hh
		if b.collChecker(px-hw*0.4, ey) ||
			b.collChecker(px, ey) ||
			b.collChecker(px+hw*0.4, ey) {
			b.velocity.y = -b.velocity.y
		}
	}
}

func (b *Barn) GateCenterY() float64 {
	return b.y + 1.5*barnGateOffY*b.drawnH
}

type CollisionOpts struct {
	isLeft bool
}

func safeSpawnPosition(collides CollisionChecker, collOpts *CollisionOpts) Vector2D {
	margin := 0.2
	var x, y float64

	for {
		if collOpts != nil {
			if collOpts.isLeft {
				x = margin*screenWidth + rand.Float64()*(screenWidth*0.5-margin*screenWidth)
			} else {
				x = screenWidth*0.5 + rand.Float64()*(screenWidth*(0.5-margin))
			}
		} else {
			x = margin*screenWidth + rand.Float64()*(screenWidth*(1-2*margin))
		}
		y = margin*screenHeight + rand.Float64()*(screenHeight*(1-2*margin))

		if !collides(x, y) &&
			!collides(x+targetBoidSize/2, y) &&
			!collides(x-targetBoidSize/2, y) &&
			!collides(x, y+targetBoidSize/2) &&
			!collides(x, y-targetBoidSize/2) {
			return Vector2D{x, y}
		}
	}
}
