package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type EventItem struct {
	position       Vector2D
	w, h           float64
	scaleX, scaleY float64
	img            *ebiten.Image
	sheet          *ebiten.Image
	eventType      EventType
	isLeft         bool

	sparkleEffect *SparkleEffect
}

func NewEventItem(position Vector2D, eventType EventType) *EventItem {
	var img *ebiten.Image
	var sheet *ebiten.Image
	switch eventType {
	case BananaEvent:
		img = bananaEventSheet
		break
	case BananaPeel:
		img = bananaPeelSheet
		break
	case EnergyEvent:
		img = energySheet
		break
	case RamEvent:
		r := ramRects[RamAttack3]
		rect := image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
		img = ramSheet.SubImage(rect).(*ebiten.Image)
		sheet = ramSheet
		break
	default:
		panic("unknown eventType")
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	scaleX := eventSize / w
	scaleY := eventSize / h

	var se *SparkleEffect
	if eventType == BananaEvent || eventType == EnergyEvent {
		se = NewSparkleEffect(eventSize, 0.95)
	}

	return &EventItem{
		position:  position,
		w:         w,
		h:         h,
		scaleX:    scaleX,
		scaleY:    scaleY,
		img:       img,
		sheet:     sheet,
		eventType: eventType,

		sparkleEffect: se,
	}
}

func (ei *EventItem) Update(_ *Player) {
	if ei.sparkleEffect != nil {
		ei.sparkleEffect.Update()
	}
}

func (ei *EventItem) IsCollidingWith(p *Player) bool {
	bHW, bHH, bOffY := ei.getCollisionBox()
	bx := ei.position.x
	by := ei.position.y + bOffY

	pHW, pHH, pOffY := p.getCollisionBox()
	px := p.position.x
	py := p.position.y + pOffY

	return math.Abs(px-bx) <= pHW+bHW &&
		math.Abs(py-by) <= pHH+bHH
}

func (ei *EventItem) DrawPickUp(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	op.GeoM.Translate(-float64(ei.w)/2, -float64(ei.h)/2)
	op.GeoM.Scale(ei.scaleX, ei.scaleY)
	op.GeoM.Translate(ei.position.x, ei.position.y)
	screen.DrawImage(ei.img, op)

	if ei.sparkleEffect != nil {
		ei.sparkleEffect.Draw(screen, ei.position.x, ei.position.y)
	}

	if isDebugMode {
		ei.drawCollisionBox(screen)
	}
}

func (ei *EventItem) GetPosition() Vector2D {
	return ei.position
}
func (ei *EventItem) IsDone() bool {
	return false
}
func (ei *EventItem) IsLeft() bool {
	return ei.isLeft
}
func (ei *EventItem) SetLeft(dir bool) {
	ei.isLeft = dir
}
func (ei *EventItem) EventType() EventType {
	return ei.eventType
}
func (ei *EventItem) ProgressState() {}

func (ei *EventItem) drawCollisionBox(screen *ebiten.Image) {
	hw, hh, offsetY := ei.getCollisionBox()
	currX := ei.position.x
	currY := ei.position.y + offsetY
	x := float32(currX - hw)
	y := float32(currY - hh)

	sideColor := func(dir Direction) color.RGBA {
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

func (ei *EventItem) getCollisionBox() (hw, hh, offsetY float64) {
	switch ei.eventType {
	case BananaEvent:
		hw = float64(ei.w) * ei.scaleX * bananaCollisionW
		hh = float64(ei.h) * ei.scaleY * bananaCollisionH
		offsetY = hh * collisionOffsetY
	case BananaPeel:
		hw = float64(ei.w) * ei.scaleX * bananaPeelCollisionW
		hh = float64(ei.h) * ei.scaleY * bananaPeelCollisionH
		offsetY = hh * collisionOffsetY
	case EnergyEvent:
		hw = float64(ei.w) * ei.scaleX * bananaCollisionW
		hh = float64(ei.h) * ei.scaleY * bananaCollisionH
		offsetY = 0
	case RamEvent:
		hw = float64(ei.w) * ei.scaleX * bananaCollisionW
		hh = float64(ei.h) * ei.scaleY * bananaCollisionH
		offsetY = 0
	}

	return
}
