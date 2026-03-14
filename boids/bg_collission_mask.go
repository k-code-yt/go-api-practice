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

func NewBgCollisionMask(img image.Image) *BgCollisionMask {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	mask := make([][]bool, h)
	for y := range h {
		mask[y] = make([]bool, w)
		for x := range w {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8
			isBush := g8 > 80 && g8 > r8+20 && g8 > b8+30
			mask[y][x] = isBush
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
