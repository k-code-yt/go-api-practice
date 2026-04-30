package main

import (
	"image"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ── Config ────────────────────────────────────────────────────────────────────
const (
	numberSheetPath = "./assets/tiles/numbers.png"
	numberSheetCols = 4
	numberSheetRows = 3
	scoreDigitH     = 62.0
	scoreDigitGap   = 4.0
	scoreSheepH     = 85.0
	scoreSheepGap   = 8.0

	scoreSheepPath       = "./assets/sheep/sheep_beee.png"
	scoreSheepFrameCount = 9  // ← 576 / 64 = 9 frames exactly
	scoreSheepFrameW     = 64 // ← exact pixel width per frame
	scoreSheepFrameDelay = 8
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

var numberSheet *ebiten.Image
var digitFrames [10]*ebiten.Image

var scoreSheepFrames [scoreSheepFrameCount]*ebiten.Image
var scoreSheepDrawW float64 // fixed at init, never changes
var scoreSheepTick int
var scoreSheepIdx int

func initScoreDisplay() {
	// ── Number sheet ──────────────────────────────────────────────────────────
	img, _, err := ebitenutil.NewImageFromFile(numberSheetPath)
	if err != nil {
		panic("number sheet: " + err.Error())
	}
	numberSheet = img
	for ch, rect := range numberRects {
		d := int(ch - '0')
		digitFrames[d] = numberSheet.SubImage(rect).(*ebiten.Image)
	}

	// ── Score sheep — load once, slice into exactly 9 frames ─────────────────
	scoreSheet, _, err := ebitenutil.NewImageFromFile(scoreSheepPath)
	if err != nil {
		// Fallback: use boid sheep sheet frame 0 for all slots
		si := NewSheepImage(sheepSheet, 5)
		f := si.Frame(0)
		for i := range scoreSheepFrames {
			scoreSheepFrames[i] = f
		}
		scoreSheepDrawW = float64(f.Bounds().Dx()) * (scoreSheepH / float64(f.Bounds().Dy()))
		return
	}

	sheetH := scoreSheet.Bounds().Dy()

	for i := 0; i < scoreSheepFrameCount; i++ {
		x0 := i * scoreSheepFrameW
		rect := image.Rect(x0, 0, x0+scoreSheepFrameW, sheetH)
		scoreSheepFrames[i] = scoreSheet.SubImage(rect).(*ebiten.Image)
	}

	// Width is constant across all frames — compute once from the fixed frameW.
	scoreSheepDrawW = float64(scoreSheepFrameW) * (scoreSheepH / float64(sheetH))
}

// UpdateScoreAnim advances the sheep animation. Call once per game Update tick.
func UpdateScoreAnim() {
	scoreSheepTick++
	if scoreSheepTick >= scoreSheepFrameDelay {
		scoreSheepTick = 0
		scoreSheepIdx = (scoreSheepIdx + 1) % scoreSheepFrameCount
	}
}

// ── Measurement ───────────────────────────────────────────────────────────────

func digitsDrawW(s string) float64 {
	w := 0.0
	for i, ch := range s {
		d := int(ch - '0')
		frame := digitFrames[d]
		if frame == nil {
			continue
		}
		db := frame.Bounds()
		w += float64(db.Dx()) * (scoreDigitH / float64(db.Dy()))
		if i < len(s)-1 {
			w += scoreDigitGap
		}
	}
	return w
}

func measureScoreSprite(score int) float64 {
	return scoreSheepDrawW + scoreSheepGap + digitsDrawW(strconv.Itoa(score))
}

// ── Draw ──────────────────────────────────────────────────────────────────────

// drawScoreSprite draws:  🐑 NUMBER  (left barn).
func drawScoreSprite(screen *ebiten.Image, score int, x, y float64) {
	frame := scoreSheepFrames[scoreSheepIdx]
	sb := frame.Bounds()
	scale := scoreSheepH / float64(sb.Dy())

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(frame, op)

	drawDigits(screen, strconv.Itoa(score), x+scoreSheepDrawW+scoreSheepGap, y)
}

// drawScoreSpriteRight draws:  NUMBER 🐑  (right barn — sheep flipped to face inward).
func drawScoreSpriteRight(screen *ebiten.Image, score int, x, y float64) {
	digits := strconv.Itoa(score)
	dw := digitsDrawW(digits)
	drawDigits(screen, digits, x, y)

	frame := scoreSheepFrames[scoreSheepIdx]
	sb := frame.Bounds()
	scale := scoreSheepH / float64(sb.Dy())
	cx := x + dw + scoreSheepGap

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(-scale, scale)
	op.GeoM.Translate(cx+scoreSheepDrawW, y)
	screen.DrawImage(frame, op)
}

// drawDigits renders digit sprites left-to-right from (cx, y).
func drawDigits(screen *ebiten.Image, digits string, cx, y float64) {
	for _, ch := range digits {
		d := int(ch - '0')
		frame := digitFrames[d]
		if frame == nil {
			continue
		}
		db := frame.Bounds()
		scale := scoreDigitH / float64(db.Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(cx, y)
		screen.DrawImage(frame, op)
		cx += float64(db.Dx())*scale + scoreDigitGap
	}
}
