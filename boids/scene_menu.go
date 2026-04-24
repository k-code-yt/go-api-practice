package main

import "github.com/hajimehoshi/ebiten/v2"

var (
	keySpace, keyEnter, keyDown, keyUp = ebiten.KeySpace, ebiten.KeyEnter, ebiten.KeyArrowDown, ebiten.KeyArrowUp
)

// menu -> start, exit, settings -> menu item
// update -> arrowup/down &&
// --- on enter/space return different sceneID
type menuItem struct {
	label   string
	sceneID SceneID
}

type MenuScene struct {
	font     *FontBitMap
	items    []menuItem
	selected SceneID
}

func NewMenuScene(font *FontBitMap) *MenuScene {
	return &MenuScene{
		font: font,
		items: []menuItem{
			{"START", ScenePlay},
			// {"EXIT", SceneExit},
		},
		selected: SceneMenu,
	}
}

func (m *MenuScene) Update() SceneID {
	itemsLen := len(m.items)

	if ebiten.IsKeyPressed(keyDown) {
		next := m.selected + 1
		if next < SceneID(itemsLen-1) {
			m.selected += 1
		}
	}

	if ebiten.IsKeyPressed(keyUp) {
		next := m.selected - 1
		if next > 0 {
			m.selected -= 1
		}
	}

	if ebiten.IsKeyPressed(keySpace) || ebiten.IsKeyPressed(keyEnter) {
		return m.selected
	}

	return SceneMenu
}

func (m *MenuScene) Draw(screen *ebiten.Image) {
	m.DrawBG(screen)
	for _, item := range m.items {
		m.font.Draw(screen, item.label)
	}
}

func (m *MenuScene) DrawBG(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bgScaleX, bgScaleY)
	screen.DrawImage(bgImage, op)
}
