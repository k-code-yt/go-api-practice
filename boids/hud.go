package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

func DrawHUD(screen *ebiten.Image, activeCount int) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.2f", ebiten.ActualFPS()))

	label := fmt.Sprintf("Sheep: %d / %d", activeCount, boidsCount)

	face := text.NewGoXFace(basicfont.Face7x13)
	charWidth := 7
	padding := 10
	x := float64(screenWidth) - float64(len(label)*charWidth) - float64(padding)

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, 2)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, label, face, op)
}
