package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	rowFront = 0
	rowSide  = 1
	rowSlip  = 2
)

type Direction int

const (
	DirDown Direction = iota
	DirLeft
	DirRight
	DirUp
)

var dirRow = map[Direction]int{
	DirDown:  rowFront,
	DirUp:    rowFront,
	DirRight: rowSide,
	DirLeft:  rowSide,
}

type PlayerState int

const (
	PlayerStateNormal   PlayerState = iota
	PlayerStateEnergy   PlayerState = iota
	PlayerStateSlipping PlayerState = iota
)

type PlayerCollisionBox struct {
	hw, hh, offsetY float64
}

type Player struct {
	position  Vector2D
	dir       Direction
	frameIdx  int
	frameTick int

	sheet     *ebiten.Image
	frameW    int
	frameH    int
	scaleX    float64
	scaleY    float64
	isMoving  bool
	isLeft    bool
	wasMoving bool
	collBox   *PlayerCollisionBox

	// collision
	collChecker CollisionChecker
	collishMap  map[Direction]bool

	// slip logic
	state    PlayerState
	slipTick int

	// under energy event
	energyTick int
}

func NewPlayer(collChecker CollisionChecker, isLeft bool) *Player {
	bounds := playerSheet.Bounds()
	fw := bounds.Dx() / playerSheetCols
	fh := bounds.Dy() / playerSheetRows

	scaleX := playerSizeX / float64(fw)
	scaleY := playerSizeY / float64(fh)
	var position Vector2D
	if isLeft {
		position = Vector2D{screenWidth / 4, screenHeight / 2}
	} else {
		position = Vector2D{screenWidth * 3 / 4, screenHeight / 2}
	}

	p := &Player{
		position:    position,
		isLeft:      isLeft,
		dir:         DirDown,
		sheet:       playerSheet,
		frameW:      fw,
		frameH:      fh,
		scaleX:      scaleX,
		scaleY:      scaleY,
		frameIdx:    0,
		collChecker: collChecker,
		collishMap:  make(map[Direction]bool),
	}

	return p
}

func (p *Player) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	sx := p.scaleX
	if p.dir == DirRight {
		sx = -p.scaleX
	}

	op.GeoM.Translate(-float64(p.frameW)/2, -float64(p.frameH)/2)
	op.GeoM.Scale(sx, p.scaleY)
	op.GeoM.Translate(p.position.x, p.position.y)
	screen.DrawImage(p.currentFrame(), op)
	if isDebugMode {
		p.drawCollisionBox(screen)
	}
}

func (p *Player) Update() {
	pSpeed := playerDefaultSpeed
	switch p.state {
	case PlayerStateSlipping:
		p.slipTick++
		if p.slipTick >= playerSlipDuration {
			p.state = PlayerStateNormal
			p.slipTick = 0
		}
		return
	case PlayerStateEnergy:
		pSpeed = playerDefaultSpeed * playerEnergyMult
		p.energyTick++
		if p.energyTick >= playerEnergyDuration {
			p.state = PlayerStateNormal
			p.energyTick = 0
		}
		break
	}

	p.UpdateMovement(pSpeed)
}

func (p *Player) UpdateMovement(pSpeed float64) {
	dx, dy := 0.0, 0.0
	p.isMoving = false

	var keyLeft, keyRight, keyDown, keyUp ebiten.Key
	if p.isLeft {
		keyLeft, keyRight, keyDown, keyUp = ebiten.KeyA, ebiten.KeyD, ebiten.KeyS, ebiten.KeyW
	} else {
		keyLeft, keyRight, keyDown, keyUp = ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyArrowDown, ebiten.KeyArrowUp
	}

	if ebiten.IsKeyPressed(keyLeft) {
		dx = -pSpeed
		p.dir = DirLeft
		p.isMoving = true
	} else if ebiten.IsKeyPressed(keyRight) {
		dx = pSpeed
		p.dir = DirRight
		p.isMoving = true
	}

	if ebiten.IsKeyPressed(keyUp) {
		dy = -pSpeed
		if !p.isMoving {
			p.dir = DirUp
		}
		p.isMoving = true
	} else if ebiten.IsKeyPressed(keyDown) {
		dy = pSpeed
		if !p.isMoving {
			p.dir = DirDown
		}
		p.isMoving = true
	}

	p.detectCollision(dx, dy)

	if p.isMoving {
		if !p.wasMoving {
			p.frameIdx = 1
			p.frameTick = 0
		} else {
			p.frameTick++
			if p.frameTick >= playerFrameDelay {
				p.frameTick = 0
				p.frameIdx = (p.frameIdx + 1) % playerSheetCols
			}
		}
	} else {
		p.frameIdx = 0
		p.frameTick = 0
		p.dir = DirDown
	}
	p.wasMoving = p.isMoving
}
func (p *Player) Energy() {
	if p.state == PlayerStateEnergy {
		return
	}
	p.state = PlayerStateEnergy
	p.energyTick = 0
}

func (p *Player) Slip() {
	if p.state == PlayerStateSlipping {
		return
	}
	p.state = PlayerStateSlipping
	p.slipTick = 0
}

func (p *Player) currentFrame() *ebiten.Image {
	col := p.frameIdx
	row := dirRow[p.dir]
	var y0 int
	if p.state == PlayerStateSlipping {
		row = rowSlip
		if p.slipTick < playerFrameDelay {
			col = 0
		} else {
			col = 1
		}
		y0 = row * (p.frameH + 10)
	} else {
		y0 = row * p.frameH
	}
	x0 := col * p.frameW
	rect := image.Rect(x0, y0, x0+p.frameW, y0+p.frameH)
	return p.sheet.SubImage(rect).(*ebiten.Image)
}

func (p *Player) detectCollision(dx, dy float64) {
	hw, hh, offsetY := p.getCollisionBox()
	nx := p.position.x + dx
	yOff := p.position.y + offsetY

	ny := p.position.y + dy
	nextYOff := ny + offsetY

	if dx > 0 {
		if !p.collChecker(nx+hw, yOff+hh) &&
			!p.collChecker(nx+hw, yOff-hh) &&
			!p.collChecker(nx+hw, yOff) {
			p.position.x = nx
			p.collishMap[DirRight] = false
		} else {
			p.collishMap[DirRight] = true
		}
		p.collishMap[DirLeft] = false
	} else {
		if !p.collChecker(nx-hw, yOff+hh) &&
			!p.collChecker(nx-hw, yOff-hh) &&
			!p.collChecker(nx-hw, yOff) {
			p.position.x = nx
			p.collishMap[DirLeft] = false
		} else {
			p.collishMap[DirLeft] = true
		}
		p.collishMap[DirRight] = false
	}

	if dy > 0 {
		if !p.collChecker(p.position.x-hw, nextYOff+hh) &&
			!p.collChecker(p.position.x, nextYOff+hh) &&
			!p.collChecker(p.position.x+hw, nextYOff+hh) {
			p.position.y = ny
			p.collishMap[DirDown] = false
		} else {
			p.collishMap[DirDown] = true
		}
		p.collishMap[DirUp] = false
	} else {
		if !p.collChecker(p.position.x-hw, nextYOff-hh) &&
			!p.collChecker(p.position.x, nextYOff-hh) &&
			!p.collChecker(p.position.x+hw, nextYOff-hh) {
			p.position.y = ny
			p.collishMap[DirUp] = false
		} else {
			p.collishMap[DirUp] = true
		}
		p.collishMap[DirDown] = false

	}
}

func (p *Player) drawCollisionBox(screen *ebiten.Image) {
	hw, hh, offsetY := p.getCollisionBox()
	currX := p.position.x
	currY := p.position.y + offsetY
	x := float32(currX - hw)
	y := float32(currY - hh)

	sideColor := func(dir Direction) color.RGBA {
		if p.collishMap[dir] {
			return color.RGBA{G: 255, A: 255}
		}
		return color.RGBA{R: 255, A: 255}
	}

	// Left edge
	vector.StrokeLine(screen, float32(x), float32(y), float32(x), float32(y+float32(hh)*2), strokeWidth, sideColor(DirLeft), false)
	// Right edge
	vector.StrokeLine(screen, float32(x+float32(hw)*2), float32(y), float32(x+float32(hw)*2), float32(y+float32(hh)*2), strokeWidth, sideColor(DirRight), false)
	// Top edge
	vector.StrokeLine(screen, float32(x), float32(y), float32(x+float32(hw)*2), float32(y), strokeWidth, sideColor(DirUp), false)
	// Bottom edge
	vector.StrokeLine(screen, float32(x), float32(y+float32(hh)*2), float32(x+float32(hw)*2), float32(y+float32(hh)*2), strokeWidth, sideColor(DirDown), false)
}

func (p *Player) getCollisionBox() (float64, float64, float64) {
	if p.collBox == nil {
		hw := float64(p.frameW) * p.scaleX * playerCollisionW
		hh := float64(p.frameH) * p.scaleY * playerCollisionH
		offsetY := hh * collisionOffsetY
		p.collBox = &PlayerCollisionBox{
			hw, hh, offsetY,
		}
	}
	return p.collBox.hw, p.collBox.hh, p.collBox.offsetY
}
