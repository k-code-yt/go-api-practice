package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type SparkleEffect struct {
	frames    [sparkleFrameCount]*ebiten.Image
	frameSize int
	frameIdx  int
	frameTick int
	eventSize float64
	scale     float64
}

func NewSparkleEffect(
	eventSize float64,
	scale float64,
) *SparkleEffect {
	s := &SparkleEffect{}

	bounds := sparkleSheet.Bounds()
	s.frameSize = bounds.Dx() / sparkleFrameCount
	fsFloat := float64(s.frameSize)

	s.scale = eventSize / fsFloat * scale
	for idx, _ := range s.frames {
		s.frames[idx] = s.currentFrame(idx)
	}
	return s
}

func (s *SparkleEffect) Update() {
	s.frameTick++
	if s.frameTick >= sparkleTicksPerFrame {
		s.frameTick = 0
		s.frameIdx = (s.frameIdx + 1) % sparkleFrameCount
	}

}

func (s *SparkleEffect) Draw(screen *ebiten.Image, x, y float64) {
	frame := s.frames[s.frameIdx]

	op := &ebiten.DrawImageOptions{}
	offset := float64(s.frameSize) * s.scale / 2

	op.GeoM.Scale(s.scale, s.scale)
	op.GeoM.Translate(x-offset, y-offset)

	screen.DrawImage(frame, op)
}

func (s *SparkleEffect) currentFrame(idx int) *ebiten.Image {
	frameWInt := int(s.frameSize)
	x0 := idx * frameWInt
	rect := image.Rect(x0, 0, x0+frameWInt, frameWInt)
	return sparkleSheet.SubImage(rect).(*ebiten.Image)
}
