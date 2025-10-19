package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func CreateClientAndDial(isClientClose bool) {
	dialer := websocket.Dialer{
		EnableCompression: true,
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  45 * time.Second,
	}

	conn, _, err := dialer.Dial("ws://localhost:3231", nil)
	if err != nil {
		log.Fatal(err)
	}
	if isClientClose {
		defer conn.Close()
	}
	time.Sleep(time.Millisecond * 100)
}

func TestConcurrentClientAdd(t *testing.T) {
	// ---SERVER SETUP
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wsSrv := NewWSServer(ctx, cancel)
	http.HandleFunc("/", wsSrv.wsHandler())
	go wsSrv.AcceptLoop()
	server := http.Server{Addr: ":3231"}
	go func() {
		err := server.ListenAndServe()
		fmt.Printf("HTTP server err = %v\n", err)
	}()
	time.Sleep(time.Millisecond * 200)
	defer server.Shutdown(ctx)
	// ----

	clientCount := 10000
	wg := new(sync.WaitGroup)
	clients := make([]*Client, clientCount)
	wg.Add(clientCount)

	for range clients {
		go func() {
			defer wg.Done()
			CreateClientAndDial(false)
		}()
	}

	wg.Wait()
	cancel()

	actualCount := wsSrv.GetClientCount()
	assert.Equal(t, 0, actualCount, "All clients left")
}
