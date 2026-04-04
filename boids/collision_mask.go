package main

type CollisionMask interface {
	IsBush(x, y float64) bool
}
