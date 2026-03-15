package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── Sprite sheet layout ──────────────────────────────────────────────────────
// 2 cols (animation frames) × 6 rows (directions / actions)
//   Row 0 → walk DOWN
//   Row 1 → walk LEFT
//   Row 2 → walk RIGHT
//   Row 3 → walk UP
//   Row 4–5 → other animations (ignored for now)

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
	DirDown  Direction = 0
	DirLeft  Direction = 1
	DirRight Direction = 2
	DirUp    Direction = 3
)

type PlayerState int

const (
	PlayerStateStaleUp    PlayerState = iota
	PlayerStateStaleDown  PlayerState = iota
	PlayerStateStaleRight PlayerState = iota
	PlayerStateStaleLeft  PlayerState = iota

	PlayerStateMovingUp    PlayerState = iota
	PlayerStateMovingDown  PlayerState = iota
	PlayerStateMovingRight PlayerState = iota
	PlayerStateMovingLeft  PlayerState = iota
)

const (
	FrameIdxStaleUpDown = 0
	FrameIdxMoveUpDown  = 1
	FrameIdxMoveLeft    = 2
	FrameIdxMoveRight   = 3
	FrameIdxStaleLeft   = 4
	FrameIdxStaleRight  = 5
)

var frameIdxToColRow = map[int][]int{}

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

	bgMask          *BgCollisionMask
	state           PlayerState
	stateToFrameMap map[PlayerState]*ebiten.Image
	nextFrame       *ebiten.Image
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
		state:    -1,
	}

	stateToFrameIdxMap := map[PlayerState]*ebiten.Image{
		PlayerStateMovingDown:  p.frame(FrameIdxMoveUpDown),
		PlayerStateMovingUp:    p.frame(FrameIdxMoveUpDown),
		PlayerStateMovingRight: p.frame(FrameIdxMoveRight),
		PlayerStateMovingLeft:  p.frame(FrameIdxMoveLeft),

		PlayerStateStaleDown:  p.frame(FrameIdxStaleUpDown),
		PlayerStateStaleUp:    p.frame(FrameIdxStaleUpDown),
		PlayerStateStaleRight: p.frame(FrameIdxStaleRight),
		PlayerStateStaleLeft:  p.frame(FrameIdxStaleLeft),
	}

	p.stateToFrameMap = stateToFrameIdxMap
	p.nextFrame = p.stateToFrameMap[FrameIdxStaleUpDown]
	return p
}

func (p *Player) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(p.frameW)/2, -float64(p.frameH)/2)
	op.GeoM.Scale(p.scaleX, p.scaleY)
	op.GeoM.Translate(p.position.x, p.position.y)
	screen.DrawImage(p.nextFrame, op)
}

func (p *Player) Update() {
	dx, dy := 0.0, 0.0
	moving := false

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		dx = -playerSpeed
		p.dir = DirLeft
		if !moving && (p.state == -1 || p.state != PlayerStateMovingLeft) {
			p.state = PlayerStateMovingLeft
		}
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
	if p.dir == DirLeft {
		p.nextFrame = p.stateToFrameMap[p.state]
		if p.state == PlayerStateMovingLeft {
			p.state = PlayerStateStaleLeft
		} else {
			p.state = PlayerStateMovingLeft
		}
	}
}

func (p *Player) frame(frameIdx int) *ebiten.Image {
	col := frameIdx % playerSheetCols // 0 or 1
	row := frameIdx / playerSheetCols
	x0 := col * p.frameW
	y0 := row * p.frameH
	rect := image.Rect(x0, y0, x0+p.frameW, y0+p.frameH)
	return p.sheet.SubImage(rect).(*ebiten.Image)
}
