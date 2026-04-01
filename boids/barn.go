package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Barn struct {
	sheet  *ebiten.Image
	flipX  bool
	scaleX float64
	scaleY float64
	x      float64
	y      float64
	w      float64
	h      float64
}

func NewBarn(flipX bool, x, y float64) *Barn {
	bounds := playerSheet.Bounds()
	w, h := float64(bounds.Dx()), float64(bounds.Dy())

	scaleX := float64(barnSizeX / w)
	scaleY := float64(barnSizeY / h)

	p := &Barn{
		w:      w,
		h:      h,
		sheet:  playerSheet,
		scaleX: scaleX,
		scaleY: scaleY,
		flipX:  flipX,
		x:      x,
		y:      y,
	}

	return p
}

func (b *Barn) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	sx := b.scaleX

	if !b.flipX {
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

func (b *Barn) drawCollisionBox(screen *ebiten.Image) {
	sign := 1.0
	if !b.flipX {
		sign = -1
	}

	boxes := []struct {
		ox, oy, w, h float64
		col          color.RGBA
	}{
		// Tune these by eye with debug rendering - roof doesn't matter
		{sign * barnBodyOffX, barnBodyOffY, barnBodyW, barnBodyH, color.RGBA{R: 255, A: 255}},
		{sign * barnFenceOffX, barnFenceOffY, barnFenceW, barnFenceH, color.RGBA{R: 255, G: 165, A: 255}},
		{sign * barnGateOffX, barnGateOffY, barnGateW, barnGateH, color.RGBA{G: 255, A: 255}},
	}
	for _, box := range boxes {
		x := float32(b.x + box.ox - box.w/2)
		y := float32(b.y + box.oy - box.h/2)
		vector.StrokeRect(screen, x, y, float32(box.w), float32(box.h), strokeWidth, box.col, false)
	}
}

func (b *Barn) IsGate(pos Vector2D) {}

func (b *Barn) IsBarnCollision(pos Vector2D) {

}

func (b *Barn) getCollisionBox(screen *ebiten.Image) {}
