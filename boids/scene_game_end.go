package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type EndGameScene struct {
	font      *FontBitMap
	items     []menuItem
	selected  int
	pulseTick int
	winner    *Player
}

func NewEndGameScene(font *FontBitMap, winner *Player) *EndGameScene {
	return &EndGameScene{
		font: font,
		items: []menuItem{
			{"GO AGAIN", ScenePlay},
			{"MAIN MENU", SceneMenu},
		},
		selected: 0,
		winner:   winner,
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

	return SceneMenu
}

func (m *EndGameScene) Draw(screen *ebiten.Image) {
	m.DrawBG(screen)

	darkGray := color.NRGBA{R: 30, G: 30, B: 30, A: 220}
	vector.FillRect(screen, 0, 0, screenWidth, screenHeight, darkGray, false)
	itemsLen := len(m.items)
	rowHight := fontScale * 1.1
	heightStep := float64(itemsLen) / 2 * rowHight
	cy := screenMiddleH + fontScale/2 - heightStep

	for idx, item := range m.items {
		scaleMult := 1.0
		if m.selected == idx {
			scaleMult = 1.0 + 0.07*math.Sin(float64(m.pulseTick)*0.15)
		}

		m.font.Draw(screen, item.label, cy, scaleMult)
		cy += rowHight
	}
}

func (m *EndGameScene) DrawBG(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bgScaleX, bgScaleY)
	screen.DrawImage(bgImage, op)
}
