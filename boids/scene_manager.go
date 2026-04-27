package main

import "github.com/hajimehoshi/ebiten/v2"

type SceneID int

const (
	SceneMenu SceneID = iota
	ScenePlay
	SceneGameEnd
	SceneExit
)

type Scene interface {
	Update() SceneID
	Draw(screen *ebiten.Image)
}

type SceneManager struct {
	currentID   SceneID
	currenScene Scene
	winSceneCH  chan *Player
}

func NewSceneManager() *SceneManager {
	return &SceneManager{
		currentID:   SceneMenu,
		currenScene: NewMenuScene(menuFontBitMap),
		winSceneCH:  make(chan *Player),
	}
}

func (s *SceneManager) Update() error {
	next := s.currenScene.Update()
	if s.currentID == next {
		return nil
	}

	var winner *Player
	select {
	case winner := <-s.winSceneCH:
		s.currenScene = NewEndGameScene(menuFontBitMap, winner)
		s.currentID = next
		return nil
	default:
	}

	switch next {
	case SceneMenu:
		s.currenScene = NewMenuScene(menuFontBitMap)
		s.currentID = next
	case ScenePlay:
		s.currenScene = NewPlayScene(s.winSceneCH)
		s.currentID = next
		break
	case SceneGameEnd:
		s.currenScene = NewEndGameScene(menuFontBitMap, winner)
		s.currentID = next
		break
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
