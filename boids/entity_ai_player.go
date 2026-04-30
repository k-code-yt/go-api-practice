package main

type AIState int

const (
	AIStateSeekSheep AIState = iota
	AIStateHerd
	AIStateSeekItem
)
