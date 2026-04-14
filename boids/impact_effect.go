package main

import "github.com/hajimehoshi/ebiten/v2"

type ImpactEffect struct {
	position      Vector2D
	frameIdx      int
	frameTick     int
	done          bool
	frameIds      []FrameID
	ticksPerFrame int
	spritesheet   *Spritesheet
	parentSizeX   float64
	parentSizeY   float64
}

func NewImpactEffect(
	pos Vector2D,
	frameIds []FrameID,
	ticksPerFrame int,
	spritesheet *Spritesheet,
	parentSizeX float64,
	parentSizeY float64,
) *ImpactEffect {
	return &ImpactEffect{
		position:      pos,
		frameIds:      frameIds,
		ticksPerFrame: ticksPerFrame,
		spritesheet:   spritesheet,
		parentSizeX:   parentSizeX,
		parentSizeY:   parentSizeY,
	}
}

func (e *ImpactEffect) Update() {
	if e.done {
		return
	}
	e.frameTick++
	if e.frameTick >= e.ticksPerFrame {
		e.frameIdx++
		e.frameTick = 0
		if e.frameIdx >= len(e.frameIds) {
			e.done = true
		}
	}
}

func (e *ImpactEffect) Draw(screen *ebiten.Image) {
	if e.done {
		return
	}

	fId := e.frameIds[e.frameIdx]
	frame := e.spritesheet.frames[fId]
	op := &ebiten.DrawImageOptions{}
	bounds := frame.Bounds()
	fw := float64(bounds.Dx())
	fh := float64(bounds.Dy())

	// 	if r.hitDir.x < 0 {
	// 	sx = -scaleX
	// }

	sx := e.parentSizeX / fw
	sy := e.parentSizeY / fh

	op.GeoM.Translate(-fw/2, -fh/2)
	op.GeoM.Scale(sx/1.5, sy/1.5)
	op.GeoM.Translate(e.position.x, e.position.y)
	screen.DrawImage(frame, op)
}
