package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ── Runtime settings ──────────────────────────────────────────────────────────

var (
	ActiveWindowScale float64 = windowScale
	ActiveBoidsCount  int     = boidsCount
	ActiveAIFlags     [2]bool = [2]bool{false, true} // [leftPlayer, rightPlayer]
)

// ── Option tables ─────────────────────────────────────────────────────────────

var scaleOptions = []struct {
	label string
	value float64
}{
	{"1X", 1.25},
	{"2X", 1.75},
	{"3X", 2.25},
}

var boidsCountOptions = []struct {
	label string
	value int
}{
	{"100", 100},
	{"150", 150},
	{"250", 250},
	{"500", 500},
}

// ── Scene ─────────────────────────────────────────────────────────────────────
type settingsRow int

const (
	rowScale settingsRow = iota
	rowBoids
	rowAI
	rowBack
	rowTotalCount
)

type SettingsScene struct {
	font      *FontBitMap
	pulseTick int
	focusRow  settingsRow
	scaleIdx  int
	boidsIdx  int
	aiFlags   [2]bool
}

func NewSettingsScene(font *FontBitMap) *SettingsScene {
	scaleIdx := 1
	for i, o := range scaleOptions {
		if o.value == ActiveWindowScale {
			scaleIdx = i
			break
		}
	}
	boidsIdx := 1
	for i, o := range boidsCountOptions {
		if o.value == ActiveBoidsCount {
			boidsIdx = i
			break
		}
	}
	return &SettingsScene{
		font:     font,
		scaleIdx: scaleIdx,
		boidsIdx: boidsIdx,
		focusRow: rowScale,
		aiFlags:  ActiveAIFlags,
	}
}

func (s *SettingsScene) Update() SceneID {
	s.pulseTick++

	if inpututil.IsKeyJustPressed(keyDown) && s.focusRow < rowTotalCount-1 {
		s.focusRow++
	}
	if inpututil.IsKeyJustPressed(keyUp) && s.focusRow > 0 {
		s.focusRow--
	}

	wentLeft := inpututil.IsKeyJustPressed(ebiten.KeyA) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)
	wentRight := inpututil.IsKeyJustPressed(ebiten.KeyD) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)

	switch s.focusRow {
	case rowScale:
		if wentLeft && s.scaleIdx > 0 {
			s.scaleIdx--
		}
		if wentRight && s.scaleIdx < len(scaleOptions)-1 {
			s.scaleIdx++
		}
	case rowBoids:
		if wentLeft && s.boidsIdx > 0 {
			s.boidsIdx--
		}
		if wentRight && s.boidsIdx < len(boidsCountOptions)-1 {
			s.boidsIdx++
		}
	case rowAI:
		if wentLeft {
			s.aiFlags[0] = !s.aiFlags[0]
		}
		if wentRight {
			s.aiFlags[1] = !s.aiFlags[1]
		}
	}

	if inpututil.IsKeyJustPressed(keySpace) || inpututil.IsKeyJustPressed(keyEnter) {
		if s.focusRow == rowBack {
			s.applySettings()
			return SceneMenu
		}
	}

	return SceneSettings
}

func (s *SettingsScene) applySettings() {
	ActiveWindowScale = scaleOptions[s.scaleIdx].value
	ActiveBoidsCount = boidsCountOptions[s.boidsIdx].value
	ActiveAIFlags = s.aiFlags
	newW := int(1080.0 * ActiveWindowScale)
	newH := int(640.0 * ActiveWindowScale)
	ebiten.SetWindowSize(newW, newH)
}

// ── Draw ──────────────────────────────────────────────────────────────────────
func (s *SettingsScene) Draw(screen *ebiten.Image) {
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Scale(bgScaleX, bgScaleY)
	screen.DrawImage(bgImage, bgOp)
	vector.FillRect(screen, 0, 0, screenWidth, screenHeight,
		color.NRGBA{R: 20, G: 20, B: 20, A: 215}, false)

	rowH := fontScale * 1.5
	startY := screenMiddleH - float64(rowTotalCount-1)*rowH/2

	s.font.Draw(screen, "SETTINGS", startY-rowH*1.8, 1.4)

	s.drawSettingRow(screen, "SIZE", scaleOptionLabels(), s.scaleIdx,
		s.focusRow == rowScale, startY+float64(rowScale)*rowH)

	s.drawSettingRow(screen, "SHEEP", boidsOptionLabels(), s.boidsIdx,
		s.focusRow == rowBoids, startY+float64(rowBoids)*rowH)

	s.drawAIRow(screen, s.focusRow == rowAI, startY+float64(rowAI)*rowH)

	backY := startY + float64(rowBack)*rowH + rowH*0.15
	backMult := 1.0
	if s.focusRow == rowBack {
		backMult = 1.0 + 0.018*math.Sin(float64(s.pulseTick)*0.15)
	}
	s.font.Draw(screen, "BACK", backY, backMult)

	divY := float32(screenHeight - fontScale*2.0)
	vector.StrokeLine(screen,
		float32(screenMiddleW-fontScale*5), divY,
		float32(screenMiddleW+fontScale*5), divY,
		1.5, color.NRGBA{R: 200, G: 160, B: 0, A: 70}, false)
	s.font.Draw(screen, "UP DN SELECT  LR CHANGE", screenHeight-fontScale*1.75, 0.32)
}

func scaleOptionLabels() []string {
	out := make([]string, len(scaleOptions))
	for i, o := range scaleOptions {
		out[i] = o.label
	}
	return out
}

func boidsOptionLabels() []string {
	out := make([]string, len(boidsCountOptions))
	for i, o := range boidsCountOptions {
		out[i] = o.label
	}
	return out
}

// drawAIRow renders a bespoke two-chip row for AI player selection.
// Left chip = Knight (player 1), Right chip = Girl (player 2).
// Each chip is independently toggled: lit = AI, dim = human.
func (s *SettingsScene) drawAIRow(screen *ebiten.Image, active bool, cy float64) {
	const textScale = 0.75
	chipPadX := fontScale * 0.50
	chipPadY := fontScale * 0.10
	chipGap := fontScale * 0.35
	labelGap := fontScale * 0.8
	label := "AI"
	labelW := float64(len(label)) * letterWidth * textScale

	labels := []string{"KNIGHT", "GIRL"}
	chipWidths := make([]float64, 2)
	for i, l := range labels {
		chipWidths[i] = s.measureMixedText(l, textScale) + chipPadX*2
	}
	totalChipsW := chipWidths[0] + chipWidths[1] + chipGap

	totalRowW := labelW + labelGap + totalChipsW
	startX := screenMiddleW - totalRowW/2

	// Label
	labelMult := textScale
	if active {
		labelMult = textScale * (1.0 + 0.012*math.Sin(float64(s.pulseTick)*0.15))
	}
	s.drawTextAt(screen, label, startX, cy, labelMult)

	chipH := fontScale*textScale + chipPadY*2
	chipX := startX + labelW + labelGap

	for i, lbl := range labels {
		on := s.aiFlags[i]
		cw := chipWidths[i]

		var chipBg, chipBorder color.NRGBA
		if on {
			chipBg = color.NRGBA{R: 190, G: 130, B: 0, A: 255}
			chipBorder = color.NRGBA{R: 255, G: 215, B: 80, A: 255}
		} else {
			chipBg = color.NRGBA{R: 50, G: 50, B: 55, A: 210}
			chipBorder = color.NRGBA{R: 100, G: 100, B: 110, A: 180}
		}

		chipTop := cy - fontScale*textScale/2 - chipPadY
		vector.FillRect(screen, float32(chipX), float32(chipTop),
			float32(cw), float32(chipH), chipBg, false)
		vector.StrokeRect(screen, float32(chipX), float32(chipTop),
			float32(cw), float32(chipH), 2.5, chipBorder, false)

		txtW := s.measureMixedText(lbl, textScale)
		tx := chipX + cw/2 - txtW/2
		s.drawMixedTextAt(screen, lbl, tx, cy, textScale)

		chipX += cw + chipGap
	}

	// Hint arrows when row is active
	if active {
		alpha := uint8(180 + 75*math.Sin(float64(s.pulseTick)*0.12))
		ac := color.NRGBA{R: 255, G: 215, B: 0, A: alpha}
		sz := float32(fontScale * textScale * 0.55)
		mid := float32(cy)

		lx := float32(startX - fontScale*0.9)
		vector.StrokeLine(screen, lx+sz, mid-sz/2, lx, mid, 3, ac, false)
		vector.StrokeLine(screen, lx, mid, lx+sz, mid+sz/2, 3, ac, false)

		rx := float32(startX + totalRowW + fontScale*0.3)
		vector.StrokeLine(screen, rx, mid-sz/2, rx+sz, mid, 3, ac, false)
		vector.StrokeLine(screen, rx+sz, mid, rx, mid+sz/2, 3, ac, false)
	}
}

// drawSettingRow renders:  LABEL  [OPT1] [OPT2] [OPT3]  all centered as a unit.
func (s *SettingsScene) drawSettingRow(
	screen *ebiten.Image,
	label string,
	options []string,
	selIdx int,
	active bool,
	cy float64,
) {
	const textScale = 0.75
	charW := letterWidth * textScale

	chipPadX := fontScale * 0.50
	chipPadY := fontScale * 0.10
	chipGap := fontScale * 0.35
	labelGap := fontScale * 0.8

	labelW := float64(len(label)) * charW

	// Measure chip widths based on mixed letter+digit rendering
	chipWidths := make([]float64, len(options))
	for i, opt := range options {
		chipWidths[i] = s.measureMixedText(opt, textScale) + chipPadX*2
	}
	totalChipsW := 0.0
	for _, cw := range chipWidths {
		totalChipsW += cw
	}
	totalChipsW += float64(len(options)-1) * chipGap

	totalRowW := labelW + labelGap + totalChipsW
	startX := screenMiddleW - totalRowW/2

	// Label
	labelMult := textScale
	if active {
		labelMult = textScale * (1.0 + 0.012*math.Sin(float64(s.pulseTick)*0.15))
	}
	s.drawTextAt(screen, label, startX, cy, labelMult)

	// Chips
	chipX := startX + labelW + labelGap
	chipH := fontScale*textScale + chipPadY*2

	for i, opt := range options {
		isSel := i == selIdx
		cw := chipWidths[i]

		chipBg := color.NRGBA{R: 50, G: 50, B: 55, A: 210}
		chipBorder := color.NRGBA{R: 100, G: 100, B: 110, A: 180}
		if isSel {
			chipBg = color.NRGBA{R: 190, G: 130, B: 0, A: 255}
			chipBorder = color.NRGBA{R: 255, G: 215, B: 80, A: 255}
		}

		chipTop := cy - fontScale*textScale/2 - chipPadY
		vector.FillRect(screen, float32(chipX), float32(chipTop),
			float32(cw), float32(chipH), chipBg, false)
		vector.StrokeRect(screen, float32(chipX), float32(chipTop),
			float32(cw), float32(chipH), 2.5, chipBorder, false)

		// Center text inside chip
		txtW := s.measureMixedText(opt, textScale)
		tx := chipX + cw/2 - txtW/2
		optMult := textScale
		if isSel && active {
			optMult = textScale * (1.0 + 0.015*math.Sin(float64(s.pulseTick)*0.15))
		}
		s.drawMixedTextAt(screen, opt, tx, cy, optMult)

		chipX += cw + chipGap
	}

	// Animated ◄ ► arrows
	if active {
		alpha := uint8(180 + 75*math.Sin(float64(s.pulseTick)*0.12))
		ac := color.NRGBA{R: 255, G: 215, B: 0, A: alpha}
		sz := float32(fontScale * textScale * 0.55)
		mid := float32(cy)

		lx := float32(startX - fontScale*0.9)
		vector.StrokeLine(screen, lx+sz, mid-sz/2, lx, mid, 3, ac, false)
		vector.StrokeLine(screen, lx, mid, lx+sz, mid+sz/2, 3, ac, false)

		rx := float32(startX + totalRowW + fontScale*0.3)
		vector.StrokeLine(screen, rx, mid-sz/2, rx+sz, mid, 3, ac, false)
		vector.StrokeLine(screen, rx+sz, mid, rx, mid+sz/2, 3, ac, false)
	}
}

func (s *SettingsScene) measureMixedText(val string, scaleMult float64) float64 {
	w := 0.0
	for _, ch := range strings.ToUpper(val) {
		if ch == ' ' {
			w += letterWidth * scaleMult
			continue
		}
		if ch >= '0' && ch <= '9' {
			d := int(ch - '0')
			frame := digitFrames[d]
			if frame != nil {
				db := frame.Bounds()
				digitScale := (fontScale * scaleMult) / float64(db.Dy())
				w += float64(db.Dx()) * digitScale
			}
		} else {
			w += letterWidth * scaleMult
		}
	}
	return w
}

func (s *SettingsScene) drawMixedTextAt(screen *ebiten.Image, val string, x, cy float64, scaleMult float64) {
	cx := x
	for _, ch := range strings.ToUpper(val) {
		if ch == ' ' {
			cx += letterWidth * scaleMult
			continue
		}
		if ch >= '0' && ch <= '9' {
			d := int(ch - '0')
			frame := digitFrames[d]
			if frame != nil {
				db := frame.Bounds()
				digitScale := (fontScale * scaleMult) / float64(db.Dy())
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(digitScale, digitScale)
				op.GeoM.Translate(cx, cy-fontScale*scaleMult/2)
				screen.DrawImage(frame, op)
				cx += float64(db.Dx()) * digitScale
			}
		} else {
			frame, ok := s.font.imageMap[ch]
			if !ok {
				cx += letterWidth * scaleMult
				continue
			}
			sc := s.font.scaleMap[ch]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(sc[0]*scaleMult, sc[1]*scaleMult)
			op.GeoM.Translate(cx, cy-fontScale*scaleMult/2)
			screen.DrawImage(frame, op)
			cx += letterWidth * scaleMult
		}
	}
}

func (s *SettingsScene) drawTextAt(screen *ebiten.Image, val string, x, cy float64, scaleMult float64) {
	cx := x
	for _, ch := range strings.ToUpper(val) {
		if ch == ' ' {
			cx += letterWidth * scaleMult
			continue
		}
		frame, ok := s.font.imageMap[ch]
		if !ok {
			cx += letterWidth * scaleMult
			continue
		}
		sc := s.font.scaleMap[ch]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc[0]*scaleMult, sc[1]*scaleMult)
		op.GeoM.Translate(cx, cy-fontScale*scaleMult/2)
		screen.DrawImage(frame, op)
		cx += letterWidth * scaleMult
	}
}
