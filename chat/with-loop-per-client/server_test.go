package withloopperclient

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var (
	host = "ws://localhost"
)

type TestConfig struct {
	clientCount    int
	wg             *sync.WaitGroup
	brMsgCount     *atomic.Int64
	targetMsgCount int
}

type TestClient struct {
	conn   *websocket.Conn
	msgCH  chan *ReqMsg
	ctx    context.Context
	mu     *sync.RWMutex
	roomID string
}

func NewTestClient(conn *websocket.Conn, ctx context.Context) *TestClient {
	return &TestClient{
		conn:  conn,
		msgCH: make(chan *ReqMsg, 64),
		ctx:   ctx,
		mu:    new(sync.RWMutex),
	}
}

func (c *TestClient) writeLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.msgCH:
			err := c.conn.WriteJSON(&msg)
			if err != nil {
				fmt.Printf("error sending msg %v\n", err)
				return
			}
		}
	}
}

func (c *TestClient) readLoop(tc *TestConfig) {
	exit := make(chan struct{})
	go func() {
		defer close(exit)
		for {
			_, b, err := c.conn.ReadMessage()
			if err != nil {
				fmt.Println("error reading msg loop => exiting readLoop")
				return
			}

			if len(b) == 1 {
				fmt.Println("received ping")
				continue
			}

			tc.brMsgCount.Add(1)
		}
	}()
	select {
	case <-c.ctx.Done():
		return
	case <-exit:
		return
	}
}

func DialServer(tc *TestConfig) *websocket.Conn {
	exit := make(chan struct{})
	dialer := websocket.DefaultDialer

	conn, _, err := dialer.Dial(fmt.Sprintf("%s%s", host, WSPort), nil)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			time.Sleep(2 * time.Second)
			if tc.targetMsgCount == int(tc.brMsgCount.Load()) {
				close(exit)
				return
			}
		}
	}()

	go func() {
		<-exit
		conn.Close()
		tc.wg.Done()
	}()

	go func() {
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if len(b) > 0 {
				tc.brMsgCount.Add(1)
			}
		}
	}()

	return conn
}

// func TestBroadcast(t *testing.T) {
// 	go CreateWSServer()
// 	ctx, cancel := context.WithCancel(context.Background())
// 	time.Sleep(1 * time.Second)
// 	clientCount := 5
// 	brCount := 2

// 	tc := TestConfig{
// 		clientCount:    clientCount,
// 		wg:             new(sync.WaitGroup),
// 		brMsgCount:     new(atomic.Int64),
// 		targetMsgCount: clientCount * brCount,
// 	}
// 	tc.wg.Add(tc.clientCount + 1)

// 	brConn := DialServer(&tc)
// 	brClient := NewTestClient(brConn, ctx)
// 	go brClient.writeLoop()

// 	for range tc.clientCount {
// 		go DialServer(&tc)
// 	}
// 	time.Sleep(1 * time.Second)

// 	for range brCount {
// 		msg := ReqMsg{
// 			MsgType: MsgType_Broadcast,
// 			Data:    "hello from tests",
// 		}
// 		brClient.msgCH <- &msg
// 	}

// 	tc.wg.Wait()
// 	cancel()

// 	time.Sleep(1 * time.Second)
// 	fmt.Println("exiting test")
// }

func JoinServer(tc *TestConfig) *websocket.Conn {
	dialer := websocket.DefaultDialer

	conn, _, err := dialer.Dial(fmt.Sprintf("%s%s", host, WSPort), nil)
	if err != nil {
		log.Fatal(err)
	}

	return conn
}

func (c *TestClient) JoinRoom(tc *TestConfig, roomID string) {
	msg := NewReqMsg(MsgType_RoomJoin, roomID, "wanna join room")
	c.msgCH <- msg
}

func (c *TestClient) LeaveRoom(tc *TestConfig, roomID string) {
	msg := NewReqMsg(MsgType_RoomLeave, roomID, "wanna leave room")
	c.msgCH <- msg
}

func (c *TestClient) SendRoomMsg(tc *TestConfig, roomID string) {
	msg := NewReqMsg(MsgType_RoomMsg, roomID, "wanna send msg to room")
	c.msgCH <- msg
}

func TestRooms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer()
	go s.CreateWSServer()
	time.Sleep(1 * time.Second)
	clientCount := 5
	brCount := 3

	tc := TestConfig{
		clientCount:    clientCount,
		wg:             new(sync.WaitGroup),
		brMsgCount:     new(atomic.Int64),
		targetMsgCount: (clientCount - 1) * brCount,
	}

	rID1 := "FIRST_ROOM"
	rID2Count := 0
	rID2 := "SECOND_ROOM"
	tc.wg.Add(tc.clientCount)
	clients := []*TestClient{}
	for range tc.clientCount {
		conn := JoinServer(&tc)
		client := NewTestClient(conn, ctx)
		client.mu.Lock()
		clients = append(clients, client)
		client.mu.Unlock()
		go client.readLoop(&tc)
		go client.writeLoop()
		rID := rID1
		if rand.IntN(10) < 5 {
			rID = rID2
			rID2Count++
		}

		client.roomID = rID
		client.JoinRoom(&tc, rID)
	}

	for {
		res := s.GetTestResult(rID1)
		fmt.Printf("test_res = %v\n", res)
		if res.clientsCount == tc.clientCount-rID2Count {
			break
		}
		time.Sleep(1 * time.Second)
	}

	for idx := range brCount {
		c := clients[idx]
		go c.SendRoomMsg(&tc, c.roomID)
		time.Sleep(1 * time.Second)
	}

	for {
		res := s.GetTestResult(rID1)
		currMsgCount := int(tc.brMsgCount.Load())
		fmt.Printf("test_res = %v, curr_msg_count = %d\n", res, currMsgCount)
		if res.clientsCount == tc.clientCount-rID2Count && currMsgCount == (tc.targetMsgCount-brCount*rID2Count) {
			break
		}
		time.Sleep(1 * time.Second)
	}

	for _, client := range clients {
		client.LeaveRoom(&tc, client.roomID)

		go func(conn *websocket.Conn) {
			time.Sleep(2 * time.Second)
			conn.Close()
			time.Sleep(300 * time.Millisecond)
			tc.wg.Done()
		}(client.conn)
	}

	tc.wg.Wait()
	cancel()

	for {
		time.Sleep(1 * time.Second)
		res1 := s.GetTestResult(rID1)
		res2 := s.GetTestResult(rID2)
		if res1.clientsCount == 0 && res2.clientsCount == 0 {
			break
		}
	}

	fmt.Println("exiting test")
}
