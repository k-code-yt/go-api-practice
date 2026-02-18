package main

import (
	"fmt"
	"testing"
	"time"
)

const (
	testDur = time.Second * 5
)

func BenchmarkMain(b *testing.B) {
	exitCH := make(chan struct{})

	go func() {
		game := NewGame()
		err := game.Run()
		if err != nil {
			b.Fatal(err)
		}
		<-exitCH
	}()

	<-time.After(testDur)
	close(exitCH)
}

func BenchmarkNewGame(b *testing.B) {
	g := NewGame()
	n := g.sg.GetNeighbours(g.boids[0])
	fmt.Println(n)
	g.sg.Clean()
}
