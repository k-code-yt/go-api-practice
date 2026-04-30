package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type SceneID int

const (
	SceneMenu SceneID = iota
	ScenePlay
	SceneSettings
	SceneGameEnd
	SceneExit
)

type WinSetter func(p *Player)

type Scene interface {
	Update() SceneID
	Draw(screen *ebiten.Image)
}

type SceneManager struct {
	currentID   SceneID
	currenScene Scene
}

func NewSceneManager() *SceneManager {
	return &SceneManager{
		currentID:   SceneMenu,
		currenScene: NewMenuScene(menuFontBitMap),
	}
}

func (s *SceneManager) WinSetter(p *Player) {
	s.currenScene = NewEndGameScene(menuFontBitMap, p)
	s.currentID = SceneGameEnd
}

func (s *SceneManager) Update() error {
	next := s.currenScene.Update()
	if s.currentID == next {
		return nil
	}

	switch next {
	case SceneMenu:
		s.currenScene = NewMenuScene(menuFontBitMap)
		s.currentID = next
	case ScenePlay:
		s.currenScene = NewPlayScene(s.WinSetter)
		s.currentID = next
	case SceneSettings:
		s.currenScene = NewSettingsScene(menuFontBitMap)
		s.currentID = next
	case SceneGameEnd:
		winner := s.currenScene.(*PlayScene).game.winner
		s.currenScene = NewEndGameScene(menuFontBitMap, winner)
		s.currentID = next
	case SceneExit:
		return ebiten.Termination
	}
	return nil
}

func (s *SceneManager) Draw(screen *ebiten.Image) {
	s.currenScene.Draw(screen)
}

func (s *SceneManager) Layout(_, _ int) (sw, sh int) {
	return screenWidth, screenHeight
}
