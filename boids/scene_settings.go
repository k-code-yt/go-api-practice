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
	ActiveWindowScale float64       = windowScale
	ActiveBoidsCount  int           = boidsCount
	ActiveCharP1      CharacterType = KnightCharacter
	ActiveCharP2      CharacterType = GirlCharacter
	ActiveAIPlayer    int           = 2 // 0=none, 1=P1, 2=P2
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

var characterOptions = []struct {
	label string
	value CharacterType
}{
	{"KNIGHT", KnightCharacter},
	{"GIRL", GirlCharacter},
	{"SCIENTIST", ScientistCharacter},
}

var aiOptions = []struct {
	label string
	value int
}{
	{"NONE", 0},
	{"P1", 1},
	{"P2", 2},
	{"BOTH", 3},
}

// ── Player colours ────────────────────────────────────────────────────────────

var (
	// P1 = blue family
	colorP1Bg     = color.NRGBA{R: 0, G: 60, B: 180, A: 220}
	colorP1Border = color.NRGBA{R: 80, G: 160, B: 255, A: 255}
	// P2 = orange family
	colorP2Bg     = color.NRGBA{R: 180, G: 80, B: 0, A: 220}
	colorP2Border = color.NRGBA{R: 255, G: 160, B: 40, A: 255}
	// both players on the same chip
	colorBothBg     = color.NRGBA{R: 110, G: 0, B: 140, A: 220}
	colorBothBorder = color.NRGBA{R: 220, G: 80, B: 255, A: 255}
)

// ── Scene rows ────────────────────────────────────────────────────────────────

type settingsRow int

const (
	rowScale settingsRow = iota
	rowBoids
	rowAI
	rowChar // single row — both players pick here simultaneously, placed last before BACK
	rowBack
	rowTotalCount
)

// ── Scene struct ──────────────────────────────────────────────────────────────

type SettingsScene struct {
	font       *FontBitMap
	pulseTick  int
	charFrames []*ebiten.Image // idle preview frame per characterOptions entry

	// shared rows (only P1 WASD navigates these — arrows do same for parity)
	focusRow settingsRow
	scaleIdx int
	boidsIdx int
	aiIdx    int

	// per-player character focus — each player navigates their own row
	// using their own keys (WASD for P1, arrows for P2)
	charIdxP1 int
	charIdxP2 int
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
	charIdxP1 := 0
	for i, o := range characterOptions {
		if o.value == ActiveCharP1 {
			charIdxP1 = i
			break
		}
	}
	charIdxP2 := 0
	for i, o := range characterOptions {
		if o.value == ActiveCharP2 {
			charIdxP2 = i
			break
		}
	}
	aiIdx := 0
	for i, o := range aiOptions {
		if o.value == ActiveAIPlayer {
			aiIdx = i
			break
		}
	}

	charFrames := make([]*ebiten.Image, len(characterOptions))
	for i, o := range characterOptions {
		charFrames[i] = characterIdleFrame(o.value)
	}

	return &SettingsScene{
		font:       font,
		charFrames: charFrames,
		scaleIdx:   scaleIdx,
		boidsIdx:   boidsIdx,
		aiIdx:      aiIdx,
		charIdxP1:  charIdxP1,
		charIdxP2:  charIdxP2,
		focusRow:   rowScale,
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (s *SettingsScene) Update() SceneID {
	s.pulseTick++

	// ── Shared row navigation (both key-sets move the shared cursor) ──────────
	p1Down := inpututil.IsKeyJustPressed(ebiten.KeyS)
	p1Up := inpututil.IsKeyJustPressed(ebiten.KeyW)
	p2Down := inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)
	p2Up := inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)

	if (p1Down || p2Down) && s.focusRow < rowTotalCount-1 {
		s.focusRow++
	}
	if (p1Up || p2Up) && s.focusRow > 0 {
		s.focusRow--
	}

	// ── Left / right inputs split by player ──────────────────────────────────
	p1Left := inpututil.IsKeyJustPressed(ebiten.KeyA)
	p1Right := inpututil.IsKeyJustPressed(ebiten.KeyD)
	p2Left := inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)
	p2Right := inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)

	// Shared rows: either player can adjust (original behaviour)
	wentLeft := p1Left || p2Left
	wentRight := p1Right || p2Right

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

	// Character row: both players move their own cursor simultaneously —
	// P1 uses A/D, P2 uses ←/→, no gating needed.
	case rowChar:
		if p1Left && s.charIdxP1 > 0 {
			s.charIdxP1--
		}
		if p1Right && s.charIdxP1 < len(characterOptions)-1 {
			s.charIdxP1++
		}
		if p2Left && s.charIdxP2 > 0 {
			s.charIdxP2--
		}
		if p2Right && s.charIdxP2 < len(characterOptions)-1 {
			s.charIdxP2++
		}
	case rowAI:
		if wentLeft && s.aiIdx > 0 {
			s.aiIdx--
		}
		if wentRight && s.aiIdx < len(aiOptions)-1 {
			s.aiIdx++
		}
	}

	// Confirm / back
	confirm := inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter)
	if confirm && s.focusRow == rowBack {
		s.applySettings()
		return SceneMenu
	}

	return SceneSettings
}

func (s *SettingsScene) applySettings() {
	ActiveWindowScale = scaleOptions[s.scaleIdx].value
	ActiveBoidsCount = boidsCountOptions[s.boidsIdx].value
	ActiveCharP1 = characterOptions[s.charIdxP1].value
	ActiveCharP2 = characterOptions[s.charIdxP2].value
	ActiveAIPlayer = aiOptions[s.aiIdx].value
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
	const charChipSize = fontScale * 2.2
	const charRowExtra = charChipSize - fontScale*1.5

	// Helper: Y position for a row, accounting for the extra char-row height.
	rowY := func(row settingsRow) float64 {
		y := screenMiddleH - float64(rowTotalCount-1)*rowH/2 + float64(row)*rowH*1.25
		if row > rowChar {
			y += charRowExtra
		}
		return y
	}

	startY := rowY(0)
	s.font.Draw(screen, "SETTINGS", startY-rowH*1.8, 1.4)

	s.drawSettingRow(screen, "SIZE", scaleOptionLabels(), s.scaleIdx,
		s.focusRow == rowScale, rowY(rowScale), -1)

	s.drawSettingRow(screen, "SHEEP", boidsOptionLabels(), s.boidsIdx,
		s.focusRow == rowBoids, rowY(rowBoids), -1)

	s.drawSettingRow(screen, "AI", aiOptionLabels(), s.aiIdx,
		s.focusRow == rowAI, rowY(rowAI), -1)

	s.drawCharRow(screen, "CHARACTER", s.focusRow == rowChar, rowY(rowChar))

	backY := rowY(rowBack) + rowH*0.3
	backMult := 1.0
	if s.focusRow == rowBack {
		backMult = 1.0 + 0.018*math.Sin(float64(s.pulseTick)*0.15)
	}
	s.font.Draw(screen, "BACK", backY, backMult)

	// Hint line
	divY := float32(screenHeight - fontScale*2.0)
	vector.StrokeLine(screen,
		float32(screenMiddleW-fontScale*5), divY,
		float32(screenMiddleW+fontScale*5), divY,
		1.5, color.NRGBA{R: 200, G: 160, B: 0, A: 70}, false)
	s.font.Draw(screen, "WS DN SELECT  AD CHANGE", screenHeight-fontScale*1.75, 0.32)
}

// drawCharRow renders the single shared character-selection row using idle
// sprite previews instead of text labels. Both players' cursors are shown
// simultaneously with distinct border colours.
func (s *SettingsScene) drawCharRow(
	screen *ebiten.Image,
	label string,
	active bool,
	cy float64,
) {
	const textScale = 0.75
	const chipSize = fontScale * 2.2 // square chip — enough room for a sprite
	const nameLabelScale = 0.32      // small name below each chip
	const chipGap = fontScale * 0.45
	const labelGap = fontScale * 0.8
	const borderW = float32(3.0)

	charW := letterWidth * textScale
	labelW := float64(len(label)) * charW
	n := len(characterOptions)
	totalChipsW := float64(n)*chipSize + float64(n-1)*chipGap
	totalRowW := labelW + labelGap + totalChipsW
	startX := screenMiddleW - totalRowW/2

	// Row label (e.g. "CHARACTER")
	labelMult := textScale
	if active {
		labelMult = textScale * (1.0 + 0.012*math.Sin(float64(s.pulseTick)*0.15))
	}
	s.drawTextAt(screen, label, startX, cy, labelMult)

	chipX := startX + labelW + labelGap
	chipTop := cy - chipSize/2

	for i, opt := range characterOptions {
		isP1 := i == s.charIdxP1
		isP2 := i == s.charIdxP2

		// ── Background ──────────────────────────────────────────────────────
		var chipBg color.NRGBA
		switch {
		case isP1 && isP2:
			chipBg = colorBothBg
		case isP1:
			chipBg = colorP1Bg
		case isP2:
			chipBg = colorP2Bg
		default:
			chipBg = color.NRGBA{R: 50, G: 50, B: 55, A: 210}
		}
		vector.FillRect(screen, float32(chipX), float32(chipTop),
			float32(chipSize), float32(chipSize), chipBg, false)

		// ── Sprite preview ──────────────────────────────────────────────────
		if frame := s.charFrames[i]; frame != nil {
			b := frame.Bounds()
			fw, fh := float64(b.Dx()), float64(b.Dy())
			// Fit inside chip with padding, preserving aspect ratio
			pad := chipSize * 0.08
			available := chipSize - pad*2
			sc := available / fh
			if fw*sc > available {
				sc = available / fw
			}
			drawW := fw * sc
			drawH := fh * sc
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(
				chipX+chipSize/2-drawW/2,
				chipTop+chipSize/2-drawH/2,
			)
			screen.DrawImage(frame, op)
		}

		// ── Outer border (P2 colour or default) ─────────────────────────────
		var outerBorder color.NRGBA
		switch {
		case isP1 && isP2:
			outerBorder = colorBothBorder
		case isP2:
			outerBorder = colorP2Border
		case isP1:
			outerBorder = colorP1Border
		default:
			outerBorder = color.NRGBA{R: 100, G: 100, B: 110, A: 180}
		}
		vector.StrokeRect(screen, float32(chipX), float32(chipTop),
			float32(chipSize), float32(chipSize), borderW, outerBorder, false)

		// ── Inner border in P1 colour when both share the chip ──────────────
		if isP1 && isP2 {
			inset := float32(5)
			vector.StrokeRect(screen,
				float32(chipX)+inset, float32(chipTop)+inset,
				float32(chipSize)-inset*2, float32(chipSize)-inset*2,
				1.5, colorP1Border, false)
		}

		// ── Character name below chip ────────────────────────────────────────
		nameY := chipTop + chipSize + fontScale*nameLabelScale*0.9
		nameW := float64(len(opt.label)) * letterWidth * nameLabelScale
		s.drawTextAt(screen, opt.label, chipX+chipSize/2-nameW/2, nameY, nameLabelScale)

		// ── P1 / P2 badges below the name ───────────────────────────────────
		badgeY := nameY + fontScale*nameLabelScale*1.1
		badgeScale := nameLabelScale * 0.9
		badgeW := letterWidth * badgeScale * 2 // "P1" or "P2" = 2 chars
		if isP1 && isP2 {
			gap := badgeW * 0.3
			s.drawTextAt(screen, "P1", chipX+chipSize/2-badgeW-gap/2, badgeY, badgeScale)
			s.drawTextAt(screen, "P2", chipX+chipSize/2+gap/2, badgeY, badgeScale)
		} else if isP1 {
			s.drawTextAt(screen, "P1", chipX+chipSize/2-badgeW/2, badgeY, badgeScale)
		} else if isP2 {
			s.drawTextAt(screen, "P2", chipX+chipSize/2-badgeW/2, badgeY, badgeScale)
		}

		chipX += chipSize + chipGap
	}

	// ── Arrows (both player colours, stacked vertically) ─────────────────────
	if active {
		alpha := uint8(180 + 75*math.Sin(float64(s.pulseTick)*0.12))
		sz := float32(fontScale * textScale * 0.55)
		mid := float32(cy)
		lx := float32(startX - fontScale*0.9)
		rx := float32(startX + totalRowW + fontScale*0.3)
		p1c := color.NRGBA{R: colorP1Border.R, G: colorP1Border.G, B: colorP1Border.B, A: alpha}
		p2c := color.NRGBA{R: colorP2Border.R, G: colorP2Border.G, B: colorP2Border.B, A: alpha}

		offset := sz * 0.55
		// P1 arrows — slightly above centre
		vector.StrokeLine(screen, lx+sz, mid-offset-sz/2, lx, mid-offset, 3, p1c, false)
		vector.StrokeLine(screen, lx, mid-offset, lx+sz, mid-offset+sz/2, 3, p1c, false)
		vector.StrokeLine(screen, rx, mid-offset-sz/2, rx+sz, mid-offset, 3, p1c, false)
		vector.StrokeLine(screen, rx+sz, mid-offset, rx, mid-offset+sz/2, 3, p1c, false)
		// P2 arrows — slightly below centre
		vector.StrokeLine(screen, lx+sz, mid+offset-sz/2, lx, mid+offset, 3, p2c, false)
		vector.StrokeLine(screen, lx, mid+offset, lx+sz, mid+offset+sz/2, 3, p2c, false)
		vector.StrokeLine(screen, rx, mid+offset-sz/2, rx+sz, mid+offset, 3, p2c, false)
		vector.StrokeLine(screen, rx+sz, mid+offset, rx, mid+offset+sz/2, 3, p2c, false)
	}
}

// drawSettingRow renders a standard single-cursor row (SIZE, SHEEP).
// pass playerNum = -1 for the default gold colour.
func (s *SettingsScene) drawSettingRow(
	screen *ebiten.Image,
	label string,
	options []string,
	selIdx int,
	active bool,
	cy float64,
	playerNum int,
) {
	const textScale = 0.75
	charW := letterWidth * textScale

	chipPadX := fontScale * 0.50
	chipPadY := fontScale * 0.10
	chipGap := fontScale * 0.35
	labelGap := fontScale * 0.8

	labelW := float64(len(label)) * charW

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

	labelMult := textScale
	if active {
		labelMult = textScale * (1.0 + 0.012*math.Sin(float64(s.pulseTick)*0.15))
	}
	s.drawTextAt(screen, label, startX, cy, labelMult)

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

		txtW := s.measureMixedText(opt, textScale)
		tx := chipX + cw/2 - txtW/2
		optMult := textScale
		if isSel && active {
			optMult = textScale * (1.0 + 0.015*math.Sin(float64(s.pulseTick)*0.15))
		}
		s.drawMixedTextAt(screen, opt, tx, cy, optMult)

		chipX += cw + chipGap
	}

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

// ── Text helpers (unchanged) ──────────────────────────────────────────────────

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

func aiOptionLabels() []string {
	out := make([]string, len(aiOptions))
	for i, o := range aiOptions {
		out[i] = o.label
	}
	return out
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
