package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type EndGameScene struct {
	font       *FontBitMap
	items      []menuItem
	selected   int
	pulseTick  int
	winner     *Player
	winnerMult float64
	sx         float64
	sy         float64
	textYOff   float64
}

func NewEndGameScene(font *FontBitMap, winner *Player) *EndGameScene {
	winnerMult := 3.0
	sx := winner.scaleX * winnerMult
	sy := winner.scaleY * winnerMult
	textYOff := screenMiddleH - sy*float64(winner.frameH)/2

	return &EndGameScene{
		font: font,
		items: []menuItem{
			{"GO AGAIN", ScenePlay},
			{"MENU", SceneMenu},
		},
		selected: 0,
		winner:   winner,
		sx:       sx,
		sy:       sy,
		textYOff: textYOff,
	}
}

func (m *EndGameScene) Update() SceneID {
	itemsLen := len(m.items)
	m.pulseTick++
	if inpututil.IsKeyJustPressed(keyDown) {
		if m.selected < itemsLen-1 {
			m.selected++
		}
	}

	if inpututil.IsKeyJustPressed(keyUp) {
		if m.selected > 0 {
			m.selected--
		}
	}

	if inpututil.IsKeyJustPressed(keySpace) || inpututil.IsKeyJustPressed(keyEnter) {
		return m.items[m.selected].sceneID
	}

	return SceneGameEnd
}

func (m *EndGameScene) Draw(screen *ebiten.Image) {

	m.drawBG(screen)

	darkGray := color.NRGBA{R: 30, G: 30, B: 30, A: 220}
	vector.FillRect(screen, 0, 0, screenWidth, screenHeight, darkGray, false)
	itemsLen := len(m.items)
	rowHight := fontScale * 1.1
	heightStep := float64(itemsLen) / 2 * rowHight
	cy := letterMiddleH - heightStep + m.textYOff

	for idx, item := range m.items {
		scaleMult := 1.0
		if m.selected == idx {
			scaleMult = 1.0 + 0.01*math.Sin(float64(m.pulseTick)*0.15)
		}

		m.font.Draw(screen, item.label, cy, scaleMult)
		cy += rowHight
	}

	m.font.Draw(screen, "WINNER!!!", letterMiddleH-m.textYOff, 1.5)
	cy += rowHight

	m.drawWinner(screen)
}

func (m *EndGameScene) drawWinner(screen *ebiten.Image) {
	p := m.winner
	op := &ebiten.DrawImageOptions{}

	p.dir = DirDown
	frame := p.spritesheet.FrameForState(DirDown, PlayerStateNormal, 0, 0)

	scaleMult := 1.0 + 5*math.Sin(float64(m.pulseTick)*0.15)

	op.GeoM.Translate(-float64(p.frameW)/2, -float64(p.frameH)/2)
	op.GeoM.Scale(m.sx, m.sy)
	op.GeoM.Translate(screenMiddleW, screenMiddleH+scaleMult)

	screen.DrawImage(frame, op)
}

func (m *EndGameScene) drawBG(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bgScaleX, bgScaleY)
	screen.DrawImage(bgImage, op)
}
