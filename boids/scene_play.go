package main

import "github.com/hajimehoshi/ebiten/v2"

type PlayScene struct {
	game *Game
}

func NewPlayScene(winFN WinSetter) *PlayScene {
	return &PlayScene{game: NewGame(winFN)}
}

func (p *PlayScene) Update() SceneID {
	p.game.Update()
	if p.game.winner != nil {
		return SceneGameEnd
	}
	return ScenePlay
}

func (p *PlayScene) Draw(screen *ebiten.Image) {
	p.game.Draw(screen)
}
