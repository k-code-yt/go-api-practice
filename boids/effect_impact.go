package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	dizzyFrames = make([]*ebiten.Image, dizzySheetCols*dizzySheetRows)
)

type ImpactEffect struct {
	position      Vector2D
	target        *Vector2D
	yOffset       float64
	frameIdx      int
	frameTick     int
	done          bool
	loop          bool
	frames        []*ebiten.Image
	ticksPerFrame int
	parentSizeX   float64
	parentSizeY   float64
	frameScale    float64
}

func NewImpactEffectPos(
	pos Vector2D,
	frameIds []FrameID,
	ticksPerFrame int,
	spritesheet *Spritesheet,
	parentSizeX float64,
	parentSizeY float64,
	frameScale float64,
) *ImpactEffect {
	frames := make([]*ebiten.Image, len(frameIds))
	for i, id := range frameIds {
		frames[i] = spritesheet.frames[id]
	}
	return &ImpactEffect{
		position:      pos,
		frames:        frames,
		ticksPerFrame: ticksPerFrame,
		parentSizeX:   parentSizeX,
		parentSizeY:   parentSizeY,
	}
}

func NewImpactEffectTarget(
	frames []*ebiten.Image,
	ticksPerFrame int,
	parentSizeX float64,
	parentSizeY float64,
	yOffset float64,
	loop bool,
	target *Vector2D,
) *ImpactEffect {
	return &ImpactEffect{
		frames:        frames,
		ticksPerFrame: ticksPerFrame,
		parentSizeX:   parentSizeX,
		parentSizeY:   parentSizeY,
		loop:          loop,
		target:        target,
		frameScale:    1,
		yOffset:       yOffset,
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
		if e.frameIdx >= len(e.frames) {
			if e.loop {
				e.frameIdx = 0
			} else {
				e.done = true
			}
		}
	}
}

func (e *ImpactEffect) Draw(screen *ebiten.Image) {
	if e.done || len(e.frames) == 0 {
		return
	}

	frame := e.frames[e.frameIdx]

	op := &ebiten.DrawImageOptions{}
	bounds := frame.Bounds()
	fw := float64(bounds.Dx())
	fh := float64(bounds.Dy())
	sx := e.parentSizeX / fw
	sy := e.parentSizeY / fh

	var drawX, drawY float64
	if e.target != nil {
		drawX = e.target.x
		drawY = e.target.y + e.yOffset
	} else {
		drawX = e.position.x
		drawY = e.position.y

	}

	op.GeoM.Translate(-fw/2, -fh/2)
	op.GeoM.Scale(sx/e.frameScale, sy/e.frameScale)
	op.GeoM.Translate(drawX, drawY)
	screen.DrawImage(frame, op)
}

func loadDizzyFrames() {
	bounds := dizzySheet.Bounds()
	fw := bounds.Dx() / dizzySheetCols
	fh := bounds.Dy() / dizzySheetRows

	idx := 0
	for col := range dizzySheetCols {
		for row := range dizzySheetRows {
			rect := image.Rect(col*fw, row*fh, fw*(col+1), fh*(row+1))
			dizzyFrames[idx] = dizzySheet.SubImage(rect).(*ebiten.Image)
			idx++
		}
	}
}
