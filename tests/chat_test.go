package tests

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type TestConfig struct {
	isClientClose  bool
	isBroadcast    bool
	msgCount       atomic.Int64
	wg             *sync.WaitGroup
	targetMsgCount int
}

type Message struct {
	RoomID  string      `json:"roomID"`
	Data    interface{} `json:"data"`
	MsgType string      `json:"type"`
}

type HealthResponse struct {
	Status string `json:"status"`
	Alloc  uint64 `json:"alloc"`
	Sys    uint64 `json:"sys"`
	NumGC  uint32 `json:"num_gc"`
}

type TestResult struct {
	healthBefore *HealthResponse
	healthAfter  *HealthResponse
	start        time.Time
	testDurr     time.Duration
}

func NewTestResult() *TestResult {
	return &TestResult{
		start: time.Now(),
	}
}

func (tr *TestResult) LogResults() {
	tr.testDurr = time.Since(tr.start)

	allocDiff := int64(tr.healthAfter.Alloc) - int64(tr.healthBefore.Alloc)
	sysDiff := int64(tr.healthAfter.Sys) - int64(tr.healthBefore.Sys)
	gcDiff := int64(tr.healthAfter.NumGC) - int64(tr.healthBefore.NumGC)

	logrus.WithFields(logrus.Fields{
		"test_duration": fmt.Sprintf("%.2fs", tr.testDurr.Seconds()),
		"alloc_diff_KB": allocDiff / 1024,
		"sys_diff_KB":   sysDiff / 1024,
		"gc_diff":       gcDiff,
		// "alloc_before":     tr.healthBefore.Alloc,
		// "alloc_after":      tr.healthAfter.Alloc,
		// "sys_before":       tr.healthBefore.Sys,
		// "sys_after":        tr.healthAfter.Sys,
		// "gc_before": tr.healthBefore.NumGC,
		// "gc_after":  tr.healthAfter.NumGC,
	}).Info("Test completed - Memory stats comparison")
}

func getHealth() *HealthResponse {
	r, err := http.Get("http://localhost:2112/health")
	if err != nil {
		panic(err)
	}

	defer r.Body.Close()
	hr := new(HealthResponse)
	err = json.NewDecoder(r.Body).Decode(hr)
	if err != nil {
		panic(err)
	}
	return hr
}

func CreateClientAndDial(cfg *TestConfig) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		EnableCompression: true,
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  45 * time.Second,
	}

	conn, _, err := dialer.Dial("ws://localhost:2112", nil)
	if err != nil {
		log.Fatal(err)
	}

	exit := make(chan struct{})

	go func() {
		<-exit
		conn.Close()
	}()

	if cfg.isBroadcast {
		go func() {
			for {
				_, b, err := conn.ReadMessage()
				if err != nil {
					fmt.Println("err read loop = ", err)
					close(exit)
					return
				}
				msg := string(b)
				if strings.Contains(msg, "close 1006") {
					fmt.Println("Conn terminated by server")
					close(exit)
					return
				}

				cfg.msgCount.Add(1)
				if int(cfg.msgCount.Load()) == cfg.targetMsgCount {
					close(exit)
					return
				}
			}
		}()
	}

	return conn, nil
}

func main() {
	tr := NewTestResult()
	tr.healthBefore = getHealth()
	broadcastsToSend := 100
	clientCount := 100
	cfg := &TestConfig{
		isClientClose:  true,
		isBroadcast:    true,
		msgCount:       atomic.Int64{},
		wg:             new(sync.WaitGroup),
		targetMsgCount: broadcastsToSend * (clientCount - 1),
	}

	brConn, err := CreateClientAndDial(cfg)
	if err != nil {
		fmt.Println("err connecting via WS = ", err)
	}
	for i := 0; i < clientCount-1; i++ {
		go CreateClientAndDial(cfg)
	}

	time.Sleep(1 * time.Second)

	for i := 0; i < broadcastsToSend; i++ {
		time.Sleep(100 * time.Millisecond)
		msg := Message{
			RoomID:  "",
			Data:    "hello from test",
			MsgType: "broadcast",
		}
		brConn.WriteJSON(&msg)
	}

loop:
	for {
		t := time.After(60 * time.Second)
		select {
		case <-t:
			fmt.Println("Exiting due to timeout")
			break loop
		default:
			time.Sleep(2 * time.Second)
			if cfg.msgCount.Load() == int64(cfg.targetMsgCount) {
				fmt.Println("TEST DONE -> reached required number of msg")
				break loop
			}
		}
	}
	tr.healthAfter = getHealth()
	tr.LogResults()
}
