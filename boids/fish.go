package main

import "github.com/hajimehoshi/ebiten/v2"

type FishImage struct {
	img                  *ebiten.Image
	w, h, scaleX, scaleY float64
}

func NewFishImage(img *ebiten.Image) *FishImage {

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	scaleX := fishTargetSize / w
	scaleY := fishTargetSize / h
	f := FishImage{
		img, w, h, scaleX, scaleY,
	}

	return &f
}
