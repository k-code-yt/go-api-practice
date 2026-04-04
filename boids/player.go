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
	wasMoving bool

	bgMask CollisionMask // was *BgCollisionMask — now the interface
	bgColl map[Direction]bool
}

func NewPlayer(bgMask CollisionMask) *Player {
	bounds := playerSheet.Bounds()
	fw := bounds.Dx() / playerSheetCols
	fh := bounds.Dy() / playerSheetRows

	return &Player{
		position: Vector2D{screenWidth / 2, screenHeight / 2},
		dir:      DirDown,
		sheet:    playerSheet,
		frameW:   fw,
		frameH:   fh,
		bgMask:   bgMask,
		scaleX:   playerSize / float64(fw),
		scaleY:   playerSize / float64(fh),
		bgColl:   make(map[Direction]bool),
	}
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
	dx, dy := 0.0, 0.0
	p.isMoving = false

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		dx = -playerSpeed
		p.dir = DirLeft
		p.isMoving = true
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		dx = playerSpeed
		p.dir = DirRight
		p.isMoving = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		dy = -playerSpeed
		if !p.isMoving {
			p.dir = DirUp
		}
		p.isMoving = true
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		dy = playerSpeed
		if !p.isMoving {
			p.dir = DirDown
		}
		p.isMoving = true
	}

	p.detectCollisionWithBGMask(dx, dy)

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

func (p *Player) currentFrame() *ebiten.Image {
	col := p.frameIdx
	row := dirRow[p.dir]
	x0 := col * p.frameW
	y0 := row * p.frameH
	return p.sheet.SubImage(image.Rect(x0, y0, x0+p.frameW, y0+p.frameH)).(*ebiten.Image)
}

func (p *Player) detectCollisionWithBGMask(dx, dy float64) {
	hw, hh, offsetY := p.getCollisionBox()
	nx := p.position.x + dx
	ny := p.position.y + dy
	yOff := p.position.y + offsetY
	nextYOff := ny + offsetY

	// Horizontal
	if dx > 0 {
		if !p.bgMask.IsBush(nx+hw, yOff-hh) &&
			!p.bgMask.IsBush(nx+hw, yOff) &&
			!p.bgMask.IsBush(nx+hw, yOff+hh) {
			p.position.x = nx
			p.bgColl[DirRight] = false
		} else {
			p.bgColl[DirRight] = true
		}
		p.bgColl[DirLeft] = false
	} else if dx < 0 {
		if !p.bgMask.IsBush(nx-hw, yOff-hh) &&
			!p.bgMask.IsBush(nx-hw, yOff) &&
			!p.bgMask.IsBush(nx-hw, yOff+hh) {
			p.position.x = nx
			p.bgColl[DirLeft] = false
		} else {
			p.bgColl[DirLeft] = true
		}
		p.bgColl[DirRight] = false
	}

	// Vertical
	if dy > 0 {
		if !p.bgMask.IsBush(p.position.x-hw, nextYOff+hh) &&
			!p.bgMask.IsBush(p.position.x, nextYOff+hh) &&
			!p.bgMask.IsBush(p.position.x+hw, nextYOff+hh) {
			p.position.y = ny
			p.bgColl[DirDown] = false
		} else {
			p.bgColl[DirDown] = true
		}
		p.bgColl[DirUp] = false
	} else if dy < 0 {
		if !p.bgMask.IsBush(p.position.x-hw, nextYOff-hh) &&
			!p.bgMask.IsBush(p.position.x, nextYOff-hh) &&
			!p.bgMask.IsBush(p.position.x+hw, nextYOff-hh) {
			p.position.y = ny
			p.bgColl[DirUp] = false
		} else {
			p.bgColl[DirUp] = true
		}
		p.bgColl[DirDown] = false
	}
}

func (p *Player) drawCollisionBox(screen *ebiten.Image) {
	hw, hh, offsetY := p.getCollisionBox()
	cx := p.position.x
	cy := p.position.y + offsetY
	x := float32(cx - hw)
	y := float32(cy - hh)

	sideColor := func(dir Direction) color.RGBA {
		if p.bgColl[dir] {
			return color.RGBA{G: 255, A: 255}
		}
		return color.RGBA{R: 255, A: 255}
	}

	vector.StrokeLine(screen, x, y, x, y+float32(hh*2), strokeWidth, sideColor(DirLeft), false)
	vector.StrokeLine(screen, x+float32(hw*2), y, x+float32(hw*2), y+float32(hh*2), strokeWidth, sideColor(DirRight), false)
	vector.StrokeLine(screen, x, y, x+float32(hw*2), y, strokeWidth, sideColor(DirUp), false)
	vector.StrokeLine(screen, x, y+float32(hh*2), x+float32(hw*2), y+float32(hh*2), strokeWidth, sideColor(DirDown), false)
}

func (p *Player) getCollisionBox() (hw, hh, offsetY float64) {
	hw = float64(p.frameW) * p.scaleX * collisionW
	hh = float64(p.frameH) * p.scaleY * collisionH
	offsetY = hh * collisionOffsetY
	return
}
