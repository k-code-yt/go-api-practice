package main

import (
	"image"
	_ "image/jpeg"
)

type BgCollisionMask struct {
	mask   [][]bool
	width  int
	height int
}

const maskScale = 6

func NewBgCollisionMask(img image.Image) *BgCollisionMask {
	bounds := img.Bounds()
	w := bounds.Dx() / maskScale
	h := bounds.Dy() / maskScale

	mask := make([][]bool, h)
	for y := range h {
		mask[y] = make([]bool, w)
		for x := range w {
			// sample center of each macro-pixel
			px := x*maskScale + maskScale/2
			py := y*maskScale + maskScale/2
			r, g, b, _ := img.At(px+bounds.Min.X, py+bounds.Min.Y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8
			mask[y][x] = g8 > 80 && g8 > r8+20 && g8 > b8+30
		}
	}
	return &BgCollisionMask{mask: mask, width: w, height: h}
}

func (cm *BgCollisionMask) IsBush(x, y float64) bool {
	mx := int(x / screenWidth * float64(cm.width))
	my := int(y / screenHeight * float64(cm.height))

	if mx < 0 || my < 0 || mx >= cm.width || my >= cm.height {
		return true
	}
	return cm.mask[my][mx]
}
