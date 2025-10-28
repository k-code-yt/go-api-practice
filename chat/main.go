package main

import (
	server "github.com/k-code-yt/go-api-practice/chat/with-loop-per-client"
)

func main() {
	// loadENV()
	// initProm()
	// chatClientLoop()
	// chatServer()
	s := server.NewServer()
	s.CreateWSServer()
}
