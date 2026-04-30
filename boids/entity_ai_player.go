package main

import "math"

const aiMiddleMargin = barnSizeX * 0.4
const aiAxisThreshold = 12.0

type AIState int

const (
	AIStateHerding AIState = iota
	AIStateSeekPickup
)

type AIPlayer struct {
	player       *Player
	ownBarn      *Barn
	boids        []*Boid
	eventManager *EventManager

	state      AIState
	targetBoid *Boid

	lastDirX float64
	lastDirY float64

	targetFleeTick int
}

func NewAIPlayer(player *Player, ownBarn *Barn, boids []*Boid, em *EventManager) *AIPlayer {
	return &AIPlayer{
		player:       player,
		ownBarn:      ownBarn,
		boids:        boids,
		eventManager: em,
		state:        AIStateHerding,
	}
}

func (ai *AIPlayer) Update() {
	p := ai.player

	if p.state == PlayerStateSlipping || p.state == PlayerStateDazed {
		p.Update()
		return
	}

	// handle active effects
	for i := len(p.activeEffects) - 1; i >= 0; i-- {
		p.activeEffects[i].Update()
		if p.activeEffects[i].done {
			p.activeEffects = append(p.activeEffects[:i], p.activeEffects[i+1:]...)
		}
	}

	pSpeed := playerDefaultSpeed
	if p.state == PlayerStateEnergy {
		pSpeed = playerDefaultSpeed * playerEnergyMult
		p.energyTick++
		if p.energyTick >= playerEnergyDuration {
			p.resetEnergy()
		} else {
			p.trailSpawnTick++
			if p.trailSpawnTick >= trailSpawnInterval {
				p.trailSpawnTick = 0
				sx := p.scaleX
				if p.dir == DirRight {
					sx = -p.scaleX
				}
				p.trailFrames = append(p.trailFrames, &TrailFrame{
					position: p.position,
					frame:    p.currentFrame(),
					scaleX:   sx,
					scaleY:   p.scaleY,
				})
				if len(p.trailFrames) >= trailMaxCount {
					p.trailFrames = p.trailFrames[1:]
				}
			}
		}
	}

	ai.updateTarget()

	var dx, dy float64
	switch ai.state {
	case AIStateSeekPickup:
		if pos := ai.energyItemPos(); pos != nil {
			dx, dy = ai.stableMoveToward(*pos, pSpeed)
		}
	case AIStateHerding:
		if ai.targetBoid != nil {
			dx, dy = ai.herdMove(pSpeed)
		}
	}

	p.isMoving = dx != 0 || dy != 0
	p.applyMovement(dx, dy, pSpeed)

	m := float64(aiMiddleMargin)
	if p.position.x < m {
		p.position.x = m
	} else if p.position.x > screenWidth-m {
		p.position.x = screenWidth - m
	}
	if p.position.y < m {
		p.position.y = m
	} else if p.position.y > screenHeight-m {
		p.position.y = screenHeight - m
	}
}

func (ai *AIPlayer) updateTarget() {
	// item always wins
	if ai.energyItemPos() != nil {
		ai.state = AIStateSeekPickup
		ai.targetBoid = nil
		ai.targetFleeTick = 0
		return
	}

	if ai.state == AIStateSeekPickup {
		ai.targetBoid = nil
	}
	ai.state = AIStateHerding

	// always snap to a sheep near the gate if one exists
	if near := ai.sheepNearGate(); near != nil && near != ai.targetBoid {
		ai.targetBoid = near
		ai.targetFleeTick = 0
	}

	if ai.targetBoid != nil {
		b := ai.targetBoid
		gate := Vector2D{ai.ownBarn.x, ai.ownBarn.GateCenterY()}

		if b.state == StateInBarn || b.state == StateMovingToDoor || !ai.inMiddleArea(b.position) {
			ai.targetBoid = nil
			ai.targetFleeTick = 0
		} else if b.position.Distance(gate) > aiTargetSwitchDist {
			ai.targetFleeTick++
			if ai.targetFleeTick >= aiTargetFleeingTicks {
				ai.targetBoid = nil
				ai.targetFleeTick = 0
			}
		} else {
			ai.targetFleeTick = 0
		}
	}

	if ai.targetBoid == nil {
		ai.targetBoid = ai.closestSheepToGate()
	}
}

func (ai *AIPlayer) herdMove(pSpeed float64) (float64, float64) {
	b := ai.targetBoid
	barnCenter := ai.ownBarn.centerPos()
	toBoid := b.position.Sub(barnCenter).Normalize()
	behindPos := b.position.Add(toBoid.Mul(targetBoidSize * 2.5))
	return ai.stableMoveToward(behindPos, pSpeed)
}

func (ai *AIPlayer) stableMoveToward(target Vector2D, pSpeed float64) (float64, float64) {
	dx := target.x - ai.player.position.x
	dy := target.y - ai.player.position.y

	var outX float64
	if math.Abs(dx) < 4 {
		outX = 0
		ai.lastDirX = 0
	} else if math.Abs(dx) >= aiAxisThreshold {
		ai.lastDirX = math.Copysign(1, dx)
		outX = ai.lastDirX * pSpeed
	} else {
		outX = ai.lastDirX * pSpeed
	}

	var outY float64
	if math.Abs(dy) < 4 {
		outY = 0
		ai.lastDirY = 0
	} else if math.Abs(dy) >= aiAxisThreshold {
		ai.lastDirY = math.Copysign(1, dy)
		outY = ai.lastDirY * pSpeed
	} else {
		outY = ai.lastDirY * pSpeed
	}

	return outX, outY
}

func (ai *AIPlayer) inMiddleArea(pos Vector2D) bool {
	m := float64(aiMiddleMargin)
	return pos.x > m && pos.x < screenWidth-m &&
		pos.y > m && pos.y < screenHeight-m
}

func (ai *AIPlayer) closestSheepToGate() *Boid {
	gate := Vector2D{ai.ownBarn.x, ai.ownBarn.GateCenterY()}
	best := math.MaxFloat64
	var found *Boid
	for _, b := range ai.boids {
		if b.state == StateInBarn || b.state == StateMovingToDoor {
			continue
		}
		if !ai.inMiddleArea(b.position) {
			continue
		}
		if d := b.position.DistanceSq(gate); d < best {
			best = d
			found = b
		}
	}
	return found
}

func (ai *AIPlayer) energyItemPos() *Vector2D {
	em := ai.eventManager
	if em.nextEvent == nil || em.nextEvent.EventType() != EnergyEvent {
		return nil
	}
	pos := em.nextEvent.GetPosition()
	if !ai.inMiddleArea(pos) {
		return nil
	}
	return &pos
}

func (b *Barn) centerPos() Vector2D {
	return Vector2D{x: b.x, y: b.y}
}

func (ai *AIPlayer) sheepNearGate() *Boid {
	gate := Vector2D{ai.ownBarn.x, ai.ownBarn.GateCenterY()}

	// horizontal band — sheep must be within barn's vertical extent
	barnHalfH := ai.ownBarn.drawnH / 2
	barnTop := ai.ownBarn.y - barnHalfH
	barnBottom := ai.ownBarn.y + barnHalfH

	var best *Boid
	bestDist := math.MaxFloat64
	for _, b := range ai.boids {
		if b.state == StateInBarn || b.state == StateMovingToDoor {
			continue
		}
		// must be horizontally aligned with barn — not above or below
		if b.position.y < barnTop || b.position.y > barnBottom {
			continue
		}
		d := b.position.DistanceSq(gate)
		if d < snapSq && d < bestDist {
			bestDist = d
			best = b
		}
	}
	return best
}
