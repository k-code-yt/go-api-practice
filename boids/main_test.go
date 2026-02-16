package main

import (
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
