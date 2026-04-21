package main

// TODO(perf) -> use everywhere to reduce this calc
type CollisionBox struct {
	hw, hh, offsetY float64
}

type FrameCoords []int

type Direction int

const (
	DirDown Direction = iota
	DirLeft
	DirRight
	DirUp
	DirSlip
	DirHit
	DirExplosion
)
