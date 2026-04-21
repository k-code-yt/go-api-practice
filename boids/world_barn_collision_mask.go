package main

import (
	"image"
	_ "image/png"
)

type BarnCollisionMask struct {
	mask   [][]bool
	width  int
	height int
}

func NewBarnCollisionMask(img image.Image) *BarnCollisionMask {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	mask := make([][]bool, h)
	for y := range h {
		mask[y] = make([]bool, w)
		for x := range w {
			_, _, _, a := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			mask[y][x] = a > 128
		}
	}

	return &BarnCollisionMask{mask: mask, width: w, height: h}
}

func (cm *BarnCollisionMask) IsSolid(
	worldX, worldY float64,
	barnX, barnY float64,
	drawnW, drawnH float64,
	flipX bool,
) bool {
	lx := worldX - (barnX - drawnW/2)
	ly := worldY - (barnY - drawnH/2)

	nx := lx / drawnW
	ny := ly / drawnH

	if flipX {
		nx = 1 - nx
	}

	if nx < 0 || ny < 0 || nx >= 1 || ny >= 1 {
		return false
	}

	mx := int(nx * float64(cm.width))
	my := int(ny * float64(cm.height))
	return cm.mask[my][mx]
}
