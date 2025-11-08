package main

import ratelimitter "github.com/k-code-yt/go-api-practice/chat/ratelimiter"

func main() {
	// loadENV()
	// initProm()
	// chatClientLoop()
	// chatServer()

	// --- WS server
	// s := server.NewServer()
	// s.CreateWSServer()
	// ------------

	s := ratelimitter.CreateServer()
	go s.RunHTTPServer()

}
