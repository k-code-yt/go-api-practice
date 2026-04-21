package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type RamState int

const (
	RamStateIdle RamState = iota
	RamStateCharging
	RamStateMoving
	RamStateHit
	RamStateSleeping
	RamStateExplosion
)

type Ram struct {
	position   Vector2D
	velocity   Vector2D
	dir        Direction
	frameIdx   int
	frameTick  int
	prevScaleX float64
	hitDir     Vector2D

	img *ebiten.Image

	target      *Player
	spritesheet *Spritesheet
	// collision
	collChecker   CollisionChecker
	impactEffects []*ImpactEffect

	sparkleEffect *SparkleEffect

	// ticks && state transition
	state         RamState
	chargeTick    int
	sleepTick     int
	hitTick       int
	explosionTick int

	hitCount int
	isDone   bool
}

func NewRam(position Vector2D, target *Player) *Ram {
	se := NewSparkleEffect(eventSize, 1.3)

	r := &Ram{
		state:         RamStateIdle,
		position:      position,
		target:        target,
		spritesheet:   NewRamSpritesheet(ramSheet),
		dir:           DirDown,
		collChecker:   target.collChecker,
		sparkleEffect: se,
	}
	return r
}

func (r *Ram) IsCollidingWith(p *Player) bool {
	return r.position.Distance(p.position) < ramHitRadius
}

func (r *Ram) DrawPickUp(screen *ebiten.Image) {
	if r.isDone {
		return
	}

	if r.state == RamStateCharging {
		vector.StrokeLine(screen, float32(r.position.x), float32(r.position.y), float32(r.target.position.x), float32(r.target.position.y), 1, color.RGBA{G: 255, A: 255}, false)
	}

	frame := r.currentFrame()
	op := &ebiten.DrawImageOptions{}
	bounds := frame.Bounds()
	fw := float64(bounds.Dx())
	fh := float64(bounds.Dy())

	scaleX := ramSizeX / fw
	sy := ramSizeY / fh
	sx := scaleX

	if r.state == RamStateHit {
		sx = r.prevScaleX
	} else if r.velocity.x > 0 {
		sx = -scaleX
	}
	prevSx := sx

	op.GeoM.Translate(-fw/2, -fh/2)
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(r.position.x, r.position.y)
	screen.DrawImage(frame, op)

	for _, e := range r.impactEffects {
		e.Draw(screen)
	}

	if r.sparkleEffect != nil {
		r.sparkleEffect.Draw(screen, r.position.x, r.position.y)
	}

	if isDebugMode {
		r.drawCollisionBox(screen, fw, fh, sx, sy)
	}
	r.prevScaleX = prevSx
}

func (r *Ram) Update(p *Player) {
	if r.isDone {
		return
	}

	switch r.state {
	case RamStateIdle:
		if r.sparkleEffect != nil {
			r.sparkleEffect.Update()
		}
		r.progressAnim(ramDirFrames[stateToDir[RamStateIdle]], 12)
		return
	case RamStateCharging:
		r.chargeTick++
		r.progressAnim(ramDirFrames[stateToDir[RamStateCharging]], 8)
		if r.chargeTick%10 == 0 {
			r.target = p
		}
		if r.chargeTick >= ramChargeDuration {
			dir := r.target.position.Sub(r.position).Normalize()
			r.velocity = dir.Mul(ramSpeed)
			r.state = RamStateMoving
			r.chargeTick = 0
			r.frameIdx = 0
			return
		}
		return
	case RamStateMoving:
		r.progressAnim(ramDirFrames[stateToDir[RamStateMoving]], 12)
		r.position = r.position.Add(r.velocity)
		if r.position.Distance(p.position) < ramHitRadius {
			p.Daze()
			r.hit()
			return
		}
		if r.detectTileCollision() {
			r.hit()
			return
		}
		return
	case RamStateHit:
		r.hitTick++
		r.frameTick++
		for i, e := range r.impactEffects {
			e.Update()
			if e.done {
				r.impactEffects = append(r.impactEffects[:i], r.impactEffects[i+1:]...)
			}
		}

		r.progressAnim(ramDirFrames[stateToDir[RamStateHit]], ramHitDuration)
		if r.hitTick >= ramHitDuration {
			r.ProgressState()
		}
		return
	case RamStateExplosion:
		r.frameTick++
		r.explosionTick++
		r.progressAnim(ramDirFrames[stateToDir[RamStateExplosion]], 12)
		if r.explosionTick >= 25 {
			r.isDone = true
		}
		return

	case RamStateSleeping:
		r.sleepTick++
		r.progressAnim(ramDirFrames[stateToDir[RamStateSleeping]], 36)
		if r.sleepTick >= ramSleepDuration {
			r.state = RamStateCharging
			r.sleepTick = 0
			r.frameIdx = 0
		}
		return
	}
}

func (r *Ram) Trigger() {
	r.state = RamStateCharging
	r.frameIdx = 0
	r.sleepTick = 0
	r.chargeTick = 0
	r.sparkleEffect = nil
}

func (r *Ram) GetPosition() Vector2D {
	return r.position
}
func (r *Ram) EventType() EventType {
	return RamEvent
}

func (r *Ram) ProgressState() {
	switch r.state {
	case RamStateIdle:
		r.Trigger()
	case RamStateHit:
		r.state = RamStateSleeping
	case RamStateSleeping:
		r.state = RamStateIdle
		r.sleepTick = 0
		r.frameIdx = 0
	}
	return
}

func (r *Ram) hit() {
	r.hitCount++
	r.hitTick = 0
	r.frameIdx = 0

	if r.hitCount >= ramMaxHitCount {
		r.state = RamStateExplosion
		return
	}

	r.state = RamStateHit
	r.hitDir = r.velocity.Normalize()
	r.velocity = Vector2D{}

	explosionPos := Vector2D{
		r.position.x + r.hitDir.x*(ramSizeX/3),
		r.position.y + r.hitDir.y*(ramSizeY/3),
	}
	explostionFrames := []FrameID{RamExplode0, RamExplode1}
	explosionEffect := NewImpactEffectPos(explosionPos,
		explostionFrames,
		ramHitDuration/len(explostionFrames),
		r.spritesheet,
		ramSizeX,
		ramSizeY,
		1.5,
	)
	r.impactEffects = append(r.impactEffects, explosionEffect)
}

func (r *Ram) sleep() {
	r.state = RamStateSleeping
	r.sleepTick = 0
	r.frameIdx = 0
	r.velocity = Vector2D{}
}

func (r *Ram) getCollisionBox(w, h, scaleX, scaleY float64) (hw, hh, offsetY float64) {
	hw = w * scaleX * bananaCollisionW
	hh = h * scaleY * bananaCollisionH
	offsetY = 0
	return
}

func (r *Ram) progressAnim(frameIds []FrameID, ticksPerFrame int) {
	r.frameTick++
	if r.frameTick >= ticksPerFrame {
		r.frameIdx = (r.frameIdx + 1) % len(frameIds)
		r.frameTick = 0
	}
}

func (r *Ram) currentFrame() *ebiten.Image {
	var frameIds []FrameID
	switch r.state {
	case RamStateIdle:
		frameIds = ramDirFrames[DirDown]
	case RamStateCharging:
		frameIds = ramDirFrames[DirUp]
	case RamStateMoving:
		frameIds = ramDirFrames[DirRight]
	case RamStateHit:
		frameIds = ramDirFrames[DirHit]
	case RamStateSleeping:
		frameIds = ramDirFrames[DirSlip]
	case RamStateExplosion:
		frameIds = ramDirFrames[DirExplosion]
	default:
		return r.defautFrame()
	}
	id := frameIds[r.frameIdx%len(frameIds)]
	return r.spritesheet.frames[id]
}

func (r *Ram) defautFrame() *ebiten.Image {
	frameIds := ramDirFrames[DirUp]
	id := frameIds[r.frameIdx%len(frameIds)]
	return r.spritesheet.frames[id]
}

func (r *Ram) detectTileCollision() bool {
	hw := targetBoidSize / 2.0
	hh := targetBoidSize / 2.0
	px, py := r.position.x, r.position.y

	// --- Horizontal: check leading X edge ---
	if r.velocity.x > 0 {
		ex := px + hw
		if r.collChecker(ex, py-hh*0.4) ||
			r.collChecker(ex, py) ||
			r.collChecker(ex, py+hh*0.4) {
			return true
		}
	} else if r.velocity.x < 0 {
		ex := px - hw
		if r.collChecker(ex, py-hh*0.4) ||
			r.collChecker(ex, py) ||
			r.collChecker(ex, py+hh*0.4) {
			return true
		}
	}

	// --- Vertical: check leading Y edge ---
	if r.velocity.y > 0 {
		ey := py + hh
		if r.collChecker(px-hw*0.4, ey) ||
			r.collChecker(px, ey) ||
			r.collChecker(px+hw*0.4, ey) {
			return true
		}
	} else if r.velocity.y < 0 {
		ey := py - hh
		if r.collChecker(px-hw*0.4, ey) ||
			r.collChecker(px, ey) ||
			r.collChecker(px+hw*0.4, ey) {
			return true
		}
	}
	return false
}

// TODO(refactor) -> move to shared?
func (r *Ram) drawCollisionBox(screen *ebiten.Image, w, h, scaleX, scaleY float64) {
	hw, hh, offsetY := r.getCollisionBox(w, h, scaleX, scaleY)
	currX := r.position.x
	currY := r.position.y + offsetY
	x := float32(currX - hw)
	y := float32(currY - hh)

	// Left edge
	vector.StrokeLine(screen, float32(x), float32(y), float32(x), float32(y+float32(hh)*2), strokeWidth, color.RGBA{G: 255, A: 255}, false)
	// Right edge
	vector.StrokeLine(screen, float32(x+float32(hw)*2), float32(y), float32(x+float32(hw)*2), float32(y+float32(hh)*2), strokeWidth, color.RGBA{G: 255, A: 255}, false)
	// Top edge
	vector.StrokeLine(screen, float32(x), float32(y), float32(x+float32(hw)*2), float32(y), strokeWidth, color.RGBA{G: 255, A: 255}, false)
	// Bottom edge
	vector.StrokeLine(screen, float32(x), float32(y+float32(hh)*2), float32(x+float32(hw)*2), float32(y+float32(hh)*2), strokeWidth, color.RGBA{G: 255, A: 255}, false)
}

func (r *Ram) IsDone() bool {
	return r.isDone
}

// just a place holder for interface
func (r *Ram) IsLeft() bool {
	return false
}
func (r *Ram) SetLeft(_ bool) {
	return
}

var ramRects = map[FrameID]FrameRect{
	// row0 y=14-100  h=86  — right-facing walk (4 frames, 5th sliver is noise)
	RamWalk0: {X: 4, Y: 14, W: 87, H: 86},
	RamWalk1: {X: 91, Y: 14, W: 87, H: 86},
	RamWalk2: {X: 178, Y: 14, W: 90, H: 86},
	RamWalk3: {X: 268, Y: 14, W: 108, H: 86},

	// row1 y=100-189  h=89  — left-facing walk (3 frames, 4th sliver is noise)
	RamWalkLeft0: {X: 5, Y: 100, W: 89, H: 89},
	RamWalkLeft1: {X: 94, Y: 100, W: 90, H: 89},
	RamWalkLeft2: {X: 184, Y: 100, W: 151, H: 89},

	// row2 y=189-268  h=79  — attack/charge (4 frames)
	RamAttack0: {X: 5, Y: 189, W: 90, H: 79},
	RamAttack1: {X: 95, Y: 189, W: 88, H: 79},
	RamAttack2: {X: 183, Y: 189, W: 90, H: 79},
	RamAttack3: {X: 273, Y: 189, W: 114, H: 79},

	// row3 y=268-333  h=65  — hit flash + recovery (4 frames)
	RamHit0: {X: 4, Y: 268, W: 110, H: 65},
	RamHit1: {X: 114, Y: 268, W: 108, H: 65},
	RamHit2: {X: 222, Y: 268, W: 89, H: 65},
	RamHit3: {X: 311, Y: 268, W: 87, H: 65},

	// row4 y=333-451  h=118  — explosion (2 real frames, rest are blank/noise)
	RamExplode0: {X: 7, Y: 333, W: 90, H: 118},
	RamExplode1: {X: 99, Y: 328, W: 120, H: 110},

	// row5 y=451-549  h=98  — small squished (1 frame)
	RamSmall: {X: 5, Y: 451, W: 236, H: 98},

	// row6 y=549-623  h=74  — down-facing walk (4 frames, 5th sliver is noise)
	RamWalkDown0: {X: 4, Y: 549, W: 92, H: 74},
	RamWalkDown1: {X: 96, Y: 549, W: 91, H: 74},
	RamWalkDown2: {X: 187, Y: 549, W: 91, H: 74},
	RamWalkDown3: {X: 278, Y: 549, W: 103, H: 74},
}

var stateToDir = map[RamState]Direction{
	RamStateIdle:     DirDown,
	RamStateCharging: DirUp,
	RamStateMoving:   DirRight,
	RamStateHit:      DirHit,
	RamStateSleeping: DirSlip,
}

// TODO -> rework dir to be separate for ram and player
var ramDirFrames = map[Direction][]FrameID{
	// RamStateIdle
	DirDown: {RamWalk0, RamWalk1, RamWalk2},
	// RamStateCharging
	DirUp: {RamHit2, RamHit0, RamAttack2, RamAttack3},
	// RamStateMoving
	DirRight: {RamWalkLeft0, RamWalkLeft1},
	// RamStateHit
	DirHit: {RamHit0, RamHit1},
	// RamStateSleeping
	DirSlip: {RamWalkDown0, RamWalkDown1, RamWalkDown2, RamWalkDown3},
	// RamStateSleeping
	DirExplosion: {RamExplode0, RamExplode1},
}
