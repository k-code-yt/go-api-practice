package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// playerSheetPath = "./assets/character/kid/kiddo_full.png"
	playerSheetPath = "./assets/character/knight_sprite.png"
	playerSheetCols = 2
	playerSheetRows = 3
	// playerSheetRows  = 2
	playerFrameDelay = 15
	playerSpeed      = 3.0
	playerSize       = targetBoidSize * 1.5

	collisionHW = 0.15
	collisionHH = 0.3
)

const (
	rowFront = 0
	rowSide  = 1
)

type Direction int

const (
	DirDown  Direction = 0
	DirLeft  Direction = 1
	DirRight Direction = 2
	DirUp    Direction = 3
)

var dirRow = map[Direction]int{
	DirDown:  rowFront,
	DirUp:    rowFront,
	DirLeft:  rowSide,
	DirRight: rowSide,
}

type Player struct {
	position  Vector2D
	dir       Direction
	moving    bool
	wasMoving bool // tracks previous frame to detect movement start
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

	return &Player{
		position: Vector2D{screenWidth / 2, screenHeight / 2},
		dir:      DirDown,
		sheet:    playerSheet,
		frameW:   fw,
		frameH:   fh,
		bgMask:   bgMask,
		scaleX:   playerSize / float64(fw),
		scaleY:   playerSize / float64(fh),
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

	if ShouldDrawCollisions == true {
		p.drawCollisionBox(screen)
	}
}

func (p *Player) Update() {
	p.moving = false
	dx, dy := p.getPosOnKeyPress()
	p.detectCollisionWithBGMask(dx, dy)
	p.updateFrame()
}

func (p *Player) getCollisionBox() (hw, hh, offsetY float64) {
	scaledW := float64(p.frameW) * p.scaleX
	scaledH := float64(p.frameH) * p.scaleY
	hw = scaledW * collisionHW
	hh = scaledH * collisionHH
	offsetY = scaledH * 0.1
	return
}

func (p *Player) drawCollisionBox(screen *ebiten.Image) {
	hw, hh, offsetY := p.getCollisionBox()

	cx := p.position.x
	cy := p.position.y + offsetY

	x := float32(cx - hw)
	y := float32(cy - hh)
	w := float32(hw * 2)
	h := float32(hh * 2)

	vector.StrokeRect(screen, x, y, w, h, 2, color.RGBA{255, 0, 0, 255}, false)
}

func (p *Player) detectCollisionWithBGMask(dx, dy float64) {
	hw, hh, offsetY := p.getCollisionBox()

	cx := p.position.x
	cy := p.position.y + offsetY

	nx := p.position.x + dx
	ny := p.position.y + dy

	ncx := nx
	ncy := ny + offsetY

	if !p.bgMask.IsBush(ncx+hw, cy-hh) &&
		!p.bgMask.IsBush(ncx+hw, cy) &&
		!p.bgMask.IsBush(ncx+hw, cy+hh) &&
		!p.bgMask.IsBush(ncx-hw, cy-hh) &&
		!p.bgMask.IsBush(ncx-hw, cy) &&
		!p.bgMask.IsBush(ncx-hw, cy+hh) {
		p.position.x = nx
	}

	if !p.bgMask.IsBush(cx-hw, ncy+hh) &&
		!p.bgMask.IsBush(cx, ncy+hh) &&
		!p.bgMask.IsBush(cx+hw, ncy+hh) &&
		!p.bgMask.IsBush(cx-hw, ncy-hh) &&
		!p.bgMask.IsBush(cx, ncy-hh) &&
		!p.bgMask.IsBush(cx+hw, ncy-hh) {
		p.position.y = ny
	}
}

func (p *Player) updateFrame() {
	if p.moving {
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

	p.wasMoving = p.moving
}

func (p *Player) getPosOnKeyPress() (float64, float64) {
	dx, dy := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		dx = -playerSpeed
		p.dir = DirLeft
		p.moving = true
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		dx = playerSpeed
		p.dir = DirRight
		p.moving = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		dy = -playerSpeed
		if !p.moving {
			p.dir = DirUp
		}
		p.moving = true
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		dy = playerSpeed
		if !p.moving {
			p.dir = DirDown
		}
		p.moving = true
	}
	return dx, dy
}

func (p *Player) currentFrame() *ebiten.Image {
	col := p.frameIdx
	row := dirRow[p.dir]
	x0 := col * p.frameW
	y0 := row * p.frameH
	rect := image.Rect(x0, y0, x0+p.frameW, y0+p.frameH)
	return p.sheet.SubImage(rect).(*ebiten.Image)
}
