package main

import "github.com/hajimehoshi/ebiten/v2"

type PickUp interface {
	IsCollidingWith(p *Player) bool
	DrawPickUp(screen *ebiten.Image)
	Update(player *Player)
	GetPosition() Vector2D
	IsLeft() bool
	IsDone() bool
	SetLeft(dir bool)
	EventType() EventType
	ProgressState()
}
