package main

import (
	"log"
)

func main() {
	game := NewGame()
	log.Fatal(game.Run())
}
