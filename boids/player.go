package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	playerSheetPath  = "./assets/character/knight_sprite.png"
	playerSheetCols  = 2
	playerSheetRows  = 3
	playerFrameCount = playerSheetCols * playerSheetRows
	playerFrameDelay = 10
	playerSpeed      = 3.0
	playerSize       = targetBoidSize * 2
)

type Direction int

const (
	DirDown  Direction = iota
	DirLeft  Direction = iota
	DirRight Direction = iota
	DirUp    Direction = iota
)

// ── Player ───────────────────────────────────────────────────────────────────

type Player struct {
	position  Vector2D
	dir       Direction
	frameIdx  int
	frameTick int

	sheet  *ebiten.Image
	frameW int
	frameH int
	scaleX float64
	scaleY float64

	bgMask *BgCollisionMask
}

func NewPlayer(bgMask *BgCollisionMask) *Player {
	bounds := playerSheet.Bounds()
	fw := bounds.Dx() / playerSheetCols
	fh := bounds.Dy() / playerSheetRows

	scaleX := playerSize / float64(fw)
	scaleY := playerSize / float64(fh)

	p := &Player{
		position: Vector2D{screenWidth / 2, screenHeight / 2},
		dir:      DirDown,
		sheet:    playerSheet,
		frameW:   fw,
		frameH:   fh,
		bgMask:   bgMask,
		scaleX:   scaleX,
		scaleY:   scaleY,
		frameIdx: 0,
	}

	return p
}

func (p *Player) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(p.frameW)/2, -float64(p.frameH)/2)
	op.GeoM.Scale(p.scaleX, p.scaleY)
	op.GeoM.Translate(p.position.x, p.position.y)
	screen.DrawImage(p.currentFrame(), op)
}

func (p *Player) Update() {
	dx, dy := 0.0, 0.0
	moving := false

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		dx = -playerSpeed
		p.dir = DirLeft
		moving = true
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		dx = playerSpeed
		p.dir = DirRight
		moving = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		dy = -playerSpeed
		p.dir = DirUp
		moving = true
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		dy = playerSpeed
		p.dir = DirDown
		moving = true
	}

	hw := float64(p.frameW) * p.scaleX / 2
	hh := float64(p.frameH) * p.scaleY / 2
	nx := p.position.x + dx
	ny := p.position.y + dy

	if !p.bgMask.IsBush(nx+hw, p.position.y) && !p.bgMask.IsBush(nx-hw, p.position.y) {
		p.position.x = nx
	}
	if !p.bgMask.IsBush(p.position.x, ny+hh) && !p.bgMask.IsBush(p.position.x, ny-hh) {
		p.position.y = ny
	}

	if moving {
		p.frameTick++
		if p.frameTick >= playerFrameDelay {
			p.frameTick = 0
			p.updateNextFrame()
		}
	} else {
		p.frameIdx = 0
		p.frameTick = 0
	}
}

func (p *Player) updateNextFrame() {
}

func (p *Player) frame(frameIdx int) *ebiten.Image {
	col := frameIdx % playerSheetCols // 0 or 1
	row := frameIdx / playerSheetCols
	x0 := col * p.frameW
	y0 := row * p.frameH
	rect := image.Rect(x0, y0, x0+p.frameW, y0+p.frameH)
	return p.sheet.SubImage(rect).(*ebiten.Image)
}
