package main

type CollisionChecker func(x, y float64) bool
type GateChecker func(x, y float64) *Barn
