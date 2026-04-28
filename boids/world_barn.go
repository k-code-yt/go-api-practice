package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Barn struct {
	sheet      *ebiten.Image
	flipX      bool
	scaleX     float64
	scaleY     float64
	x          float64
	y          float64
	w          float64
	h          float64
	drawnW     float64
	drawnH     float64
	SheepCount int
	isLeft     bool

	mask *BarnCollisionMask
}

func NewBarn(flipX bool, x, y float64, mask *BarnCollisionMask) *Barn {
	bounds := barnSheet.Bounds()
	w, h := float64(bounds.Dx()), float64(bounds.Dy())

	scaleX := float64(barnSizeX / w)
	scaleY := float64(barnSizeY / h)

	b := &Barn{
		w:      w,
		h:      h,
		sheet:  barnSheet,
		scaleX: scaleX,
		scaleY: scaleY,
		flipX:  flipX,
		x:      x,
		y:      y,
		drawnW: w * scaleX,
		drawnH: h * scaleY,
		mask:   mask,
		isLeft: flipX,
	}

	return b
}

func (b *Barn) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	sx := b.scaleX

	if b.flipX {
		sx = -b.scaleX
	}

	op.GeoM.Translate(-float64(b.w)/2, -float64(b.h)/2)
	op.GeoM.Scale(sx, b.scaleY)
	op.GeoM.Translate(b.x, b.y)

	screen.DrawImage(barnSheet, op)

	if isDebugMode {
		b.drawCollisionBox(screen)
	}
}

func (b *Barn) getCollisionBox() (cx, cy, hw, hh float64) {
	sign := 1.0
	if b.flipX {
		sign = -1
	}
	cx = b.x + sign*barnGateOffX*b.drawnW
	cy = b.y + barnGateOffY*b.drawnH
	hw = barnGateW * b.drawnW / 2
	hh = barnGateH * b.drawnH / 2
	return cx, cy, hw, hh
}

func (b *Barn) drawCollisionBox(screen *ebiten.Image) {
	cx, cy, hw, hh := b.getCollisionBox()
	vector.StrokeRect(screen,
		float32(cx-hw), float32(cy-hh),
		float32(hw*2), float32(hh*2),
		strokeWidth, color.RGBA{R: 255, A: 255}, false)
}

func (b *Barn) IsGate(x, y float64) bool {
	cx, cy, hw, hh := b.getCollisionBox()

	return x >= cx-hw && x <= cx+hw &&
		y >= cy-hh && y <= cy+hh
}

func (b *Barn) IsBlocking(x, y float64) bool {
	return b.mask.IsSolid(x, y, b.x, b.y, b.drawnW, b.drawnH, b.flipX)
}
