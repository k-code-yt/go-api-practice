package main

import (
	server "github.com/k-code-yt/go-api-practice/chat/with-loop-per-client"
)

func main() {
	// loadENV()
	// initProm()

	// --- WS server
	s := server.NewServer()
	s.CreateWSServer()
	// ------------

	// s := ratelimitter.CreateServer()
	// go s.RunHTTPServer()
}
