package main

import (
	"image"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ── Config ────────────────────────────────────────────────────────────────────
const (
	numberSheetPath = "./assets/tiles/numbers.png" // 4 cols × 3 rows
	numberSheetCols = 4
	numberSheetRows = 3
	scoreDigitH     = 62.0 // target drawn height, matches old font size
	scoreDigitGap   = 4.0  // horizontal gap between digits
	scoreSheepH     = 85.0 // sheep icon drawn height
	scoreSheepGap   = 8.0  // gap between sheep icon and first digit
)

var numberRects = map[rune]image.Rectangle{
	'1': image.Rect(37, 56, 129, 173),
	'2': image.Rect(144, 56, 236, 173),
	'3': image.Rect(251, 56, 351, 173),
	'4': image.Rect(357, 56, 449, 173),
	'5': image.Rect(37, 189, 129, 302),
	'6': image.Rect(144, 189, 236, 302),
	'7': image.Rect(251, 189, 351, 302),
	'8': image.Rect(357, 189, 449, 302),
	'9': image.Rect(37, 315, 129, 430),
	'0': image.Rect(144, 315, 236, 430),
}

// digit order in the sheet, left-to-right, top-to-bottom:
// row0: 1 2 3 4
// row1: 5 6 7 8
// row2: 9 0 ? !
var digitIndex = map[rune]int{
	'1': 0, '2': 1, '3': 2, '4': 3,
	'5': 4, '6': 5, '7': 6, '8': 7,
	'9': 8, '0': 9,
}

var numberSheet *ebiten.Image
var digitFrames [10]*ebiten.Image // 0-9
var sheepFrame0 *ebiten.Image

func initScoreDisplay() {
	img, _, err := ebitenutil.NewImageFromFile(numberSheetPath)
	if err != nil {
		panic("number sheet: " + err.Error())
	}
	numberSheet = img

	for ch, rect := range numberRects {
		d := int(ch - '0')
		digitFrames[d] = numberSheet.SubImage(rect).(*ebiten.Image)
	}

	sheepImg := NewSheepImage(sheepSheet, 5)
	sheepFrame0 = sheepImg.Frame(0)
}

// drawScoreSprite draws a sheep icon followed by the score number at (x, y).
// x is the left edge; for right-aligned scores call measureScoreSprite first.
func drawScoreSprite(screen *ebiten.Image, score int, x, y float64) {
	cx := x

	// --- sheep icon ---
	sb := sheepFrame0.Bounds()
	sheepScaleX := scoreSheepH / float64(sb.Dy())
	sheepScaleY := sheepScaleX
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sheepScaleX, sheepScaleY)
	op.GeoM.Translate(cx, y)
	screen.DrawImage(sheepFrame0, op)
	cx += float64(sb.Dx())*sheepScaleX + scoreSheepGap

	// --- digits ---
	digits := strconv.Itoa(score)
	for _, ch := range digits {
		d := int(ch - '0')
		frame := digitFrames[d]
		if frame == nil {
			continue
		}
		db := frame.Bounds()
		scaleX := scoreDigitH / float64(db.Dy())
		scaleY := scaleX
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scaleX, scaleY)
		op.GeoM.Translate(cx, y)
		screen.DrawImage(frame, op)
		cx += float64(db.Dx())*scaleX + scoreDigitGap
	}
}

// measureScoreSprite returns the total pixel width of drawScoreSprite output.
func measureScoreSprite(score int) float64 {
	sb := sheepFrame0.Bounds()
	sheepScaleX := scoreSheepH / float64(sheepFrame0.Bounds().Dy())
	w := float64(sb.Dx())*sheepScaleX + scoreSheepGap

	digits := strconv.Itoa(score)
	for _, ch := range digits {
		d := int(ch - '0')
		frame := digitFrames[d]
		if frame == nil {
			continue
		}
		db := frame.Bounds()
		scaleX := scoreDigitH / float64(db.Dy())
		w += float64(db.Dx())*scaleX + scoreDigitGap
	}
	return w
}
