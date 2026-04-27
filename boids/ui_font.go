package main

import (
	"image"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const menuFontPath = "./assets/menu/menu_font.png"

// menuFontRect holds the pixel rectangle of one glyph.
type menuFontRect struct{ X, Y, W, H int }

var menuFontRects = map[rune]menuFontRect{
	// Row 0 y=16, h=51
	'A': {X: 46, Y: 16, W: 50, H: 51},
	'B': {X: 106, Y: 16, W: 52, H: 51},
	'C': {X: 166, Y: 16, W: 55, H: 51},
	'D': {X: 230, Y: 16, W: 50, H: 51},
	'E': {X: 291, Y: 16, W: 51, H: 51},
	'F': {X: 352, Y: 16, W: 51, H: 51},
	'G': {X: 413, Y: 16, W: 51, H: 51},
	'H': {X: 474, Y: 16, W: 52, H: 51},
	'I': {X: 536, Y: 16, W: 47, H: 51},
	// Row 1 y=95, h=59
	'J': {X: 37, Y: 95, W: 50, H: 59},
	'K': {X: 98, Y: 95, W: 55, H: 59},
	'L': {X: 164, Y: 95, W: 50, H: 59},
	'M': {X: 224, Y: 95, W: 62, H: 59},
	'N': {X: 296, Y: 95, W: 52, H: 59},
	'O': {X: 356, Y: 95, W: 51, H: 59},
	'P': {X: 418, Y: 95, W: 52, H: 59},
	'Q': {X: 480, Y: 95, W: 50, H: 59},
	'R': {X: 541, Y: 95, W: 50, H: 59},
	// Row 2 y=174, h=51
	'S': {X: 24, Y: 174, W: 53, H: 51},
	'T': {X: 84, Y: 174, W: 47, H: 51},
	'U': {X: 142, Y: 174, W: 50, H: 51},
	'V': {X: 203, Y: 174, W: 51, H: 51},
	'W': {X: 265, Y: 174, W: 59, H: 51},
	'X': {X: 335, Y: 174, W: 50, H: 51},
	'Y': {X: 396, Y: 174, W: 55, H: 51},
	'Z': {X: 462, Y: 174, W: 50, H: 51},
	'?': {X: 523, Y: 174, W: 51, H: 51},
	'!': {X: 584, Y: 174, W: 20, H: 51},
}

var (
	menuFontBitMap *FontBitMap
)

const (
	fontScale     = 60.0
	screenMiddleW = float64(screenWidth / 2)
	screenMiddleH = float64(screenHeight / 2)
	letterWidth   = fontScale * 1.1
)

type FontBitMap struct {
	imageMap map[rune]*ebiten.Image
	scaleMap map[rune][2]float64
}

func NewFontBitMap() *FontBitMap {
	f := &FontBitMap{}
	f.loadFontImages()
	return f
}

func (f *FontBitMap) Draw(screen *ebiten.Image, val string, cy float64, scaleMult float64) {
	strUpper := strings.ToUpper(val)
	strWidth := letterWidth * len(strUpper)
	cx := screenMiddleW - float64(strWidth/2)
	for _, ch := range strUpper {
		frame, ok := f.imageMap[ch]
		if !ok {
			log.Fatal("letter not found")
			continue
		}
		scale := f.scaleMap[ch]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale[0]*scaleMult/0.85, scale[1]*scaleMult)
		op.GeoM.Translate(cx, cy)
		screen.DrawImage(frame, op)
		cx = cx + letterWidth
	}
}

func (f *FontBitMap) loadFontImages() {
	imageMap := make(map[rune]*ebiten.Image, len(menuFontRects))
	scaleMap := make(map[rune][2]float64, len(menuFontRects))
	for keyV, rect := range menuFontRects {
		rectV := image.Rect(rect.X, rect.Y, rect.X+rect.W, rect.Y+rect.H)
		frame := menuFontSheet.SubImage(rectV).(*ebiten.Image)
		imageMap[keyV] = frame
		bounds := frame.Bounds()
		scaleX := fontScale / bounds.Dx()
		scaleY := fontScale / bounds.Dy()
		scaleMap[keyV] = [2]float64{float64(scaleX), float64(scaleY)}
	}
	f.imageMap = imageMap
	f.scaleMap = scaleMap
}
