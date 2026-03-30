package main

import (
	"fmt"
	"testing"
	"time"
)

const (
	testDur = time.Second * 15
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

func TestModulo(t *testing.T) {
	fmt.Println((0 + 1) % playerSheetCols)
}
