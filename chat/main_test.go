package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	isClientClose bool
	isBroadcast   bool
	msgCount      atomic.Int64
	wg            *sync.WaitGroup
}

func CreateClientAndDial(cfg *TestConfig) (*websocket.Conn, error) {
	// defer cfg.wg.Done()
	dialer := websocket.Dialer{
		EnableCompression: true,
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  45 * time.Second,
	}

	conn, _, err := dialer.Dial("ws://localhost:2112", nil)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.isClientClose {
		defer conn.Close()
	}
	time.Sleep(time.Millisecond * 100)

	if cfg.isBroadcast {

		go func() {
			for {
				_, b, err := conn.ReadMessage()
				if err != nil {
					fmt.Println("err read loop = ", err)
					return
				}
				msg := string(b)
				if strings.Contains(msg, "close 1006") {
					fmt.Println("Conn terminated by server")
					return
				}
				cfg.msgCount.Add(1)
			}
		}()
	}

	return conn, nil
}

func TestConcurrentClientAdd(t *testing.T) {
	loadENV()
	os.Setenv("ENV", "test")
	// ---SERVER SETUP
	wsSrv := NewWSServer(1)
	http.HandleFunc("/", wsSrv.wsHandler())
	go wsSrv.AcceptLoop()
	server := http.Server{Addr: ":2112"}
	go func() {
		err := server.ListenAndServe()
		fmt.Printf("HTTP server err = %v\n", err)
	}()
	time.Sleep(time.Millisecond * 200)
	defer server.Shutdown(wsSrv.ctx)
	// ----
	broadcastsToSend := 1
	clientCount := 5
	cfg := &TestConfig{
		isClientClose: false,
		isBroadcast:   true,
		msgCount:      atomic.Int64{},
		wg:            new(sync.WaitGroup),
	}
	// cfg.wg.Add((clientCount - 1) * broadcastsToSend)

	brConn, err := CreateClientAndDial(cfg)
	if err != nil {
		fmt.Println("err connecting via WS = ", err)
	}
	for i := 0; i < clientCount-1; i++ {
		go CreateClientAndDial(cfg)
	}

	time.Sleep(2 * time.Second)

	for i := 0; i < broadcastsToSend; i++ {
		time.Sleep(100 * time.Millisecond)
		msg := Message{
			RoomID:  "",
			Data:    "hello from test",
			MsgType: "broadcast",
		}
		brConn.WriteJSON(&msg)
	}

	// cfg.wg.Wait()
	time.Sleep(5 * time.Second)
	msgCount := int(cfg.msgCount.Load())
	assert.Equal(t, (clientCount-1)*broadcastsToSend, msgCount, "All clients received broadcast")

	wsSrv.cancelFN()
	<-wsSrv.shutdownCH

	actualClientsCount := wsSrv.GetClientCount()
	assert.Equal(t, 0, actualClientsCount, "All clients left")
}
