package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type SheepImage struct {
	img                  *ebiten.Image
	w, h, scaleX, scaleY float64
	frameCount           int
	frameW               float64
	frameH               float64
}

func NewSheepImage(img *ebiten.Image, frameCount int) *SheepImage {

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	frameW := w / float64(frameCount)
	scaleX := targetBoidSize / frameW
	scaleY := targetBoidSize / h

	f := SheepImage{
		img:        img,
		w:          w,
		h:          h,
		scaleX:     scaleX,
		scaleY:     scaleY,
		frameW:     frameW,
		frameH:     h,
		frameCount: frameCount,
	}

	return &f
}

func (s *SheepImage) Frame(idx int) *ebiten.Image {
	frameWInt := int(s.frameW)
	x0 := idx * frameWInt
	rect := image.Rect(x0, 0, x0+frameWInt, int(s.frameH))
	return s.img.SubImage(rect).(*ebiten.Image)
}
