package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type SpriteBatch struct {
	vertices []ebiten.Vertex
	indices  []uint16
}

var sharedBatch = &SpriteBatch{
	vertices: make([]ebiten.Vertex, 0, boidsCount*4),
	indices:  make([]uint16, 0, boidsCount*6),
}

func (sb *SpriteBatch) Add(
	cx, cy float64,
	srcX, srcY, srcW, srcH int,
	sheetW, sheetH int,
	scaleX, scaleY float64,
) {
	base := uint16(len(sb.vertices))

	hw := float32(float64(srcW) * 0.5 * scaleX)
	hh := float32(float64(srcH) * 0.5 * scaleY)
	fx := float32(cx)
	fy := float32(cy)

	x0 := float32(srcX)
	y0 := float32(srcY)
	x1 := float32(srcX + srcW)
	y1 := float32(srcY + srcH)

	if hw < 0 {
		x0, x1 = x1, x0
		hw = -hw
	}
	if hh < 0 {
		y0, y1 = y1, y0
		hh = -hh
	}

	sb.vertices = append(sb.vertices,
		ebiten.Vertex{DstX: fx - hw, DstY: fy - hh, SrcX: x0, SrcY: y0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: fx + hw, DstY: fy - hh, SrcX: x1, SrcY: y0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: fx - hw, DstY: fy + hh, SrcX: x0, SrcY: y1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: fx + hw, DstY: fy + hh, SrcX: x1, SrcY: y1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	)
	sb.indices = append(sb.indices,
		base+0, base+1, base+2,
		base+1, base+3, base+2,
	)
}

func (sb *SpriteBatch) Flush(screen *ebiten.Image, sheet *ebiten.Image) {
	if len(sb.indices) == 0 {
		return
	}

	op := &ebiten.DrawTrianglesOptions{
		Filter: ebiten.FilterLinear,
	}
	screen.DrawTriangles(sb.vertices, sb.indices, sheet, op)

	sb.vertices = sb.vertices[:0]
	sb.indices = sb.indices[:0]
}
