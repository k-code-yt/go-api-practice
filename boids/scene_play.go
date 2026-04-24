package main

import "github.com/hajimehoshi/ebiten/v2"

type PlayScene struct {
	game *Game
}

func NewPlayScene() *PlayScene {
	return &PlayScene{game: NewGame()}
}

func (p *PlayScene) Update() SceneID {
	p.game.Update()
	return ScenePlay
}

func (p *PlayScene) Draw(screen *ebiten.Image) {
	p.game.Draw(screen)
}
