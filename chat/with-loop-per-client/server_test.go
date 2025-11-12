package withloopperclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var (
	host = "ws://localhost"
)

type TestConfig struct {
	clientCount       int
	wg                *sync.WaitGroup
	brMsgCount        *atomic.Int64
	targetMsgCount    int
	throttledMsgCount *atomic.Int64
	lastOffset        int
}

type TestClient struct {
	conn   *websocket.Conn
	msgCH  chan *ReqMsg
	pongCH chan [1]byte
	ctx    context.Context
	mu     *sync.RWMutex
	roomID string
}

func NewTestClient(conn *websocket.Conn, ctx context.Context) *TestClient {
	return &TestClient{
		conn:   conn,
		msgCH:  make(chan *ReqMsg, 64),
		pongCH: make(chan [1]byte, 8),
		ctx:    ctx,
		mu:     new(sync.RWMutex),
	}
}

func (c *TestClient) writeLoop() {
	defer c.conn.Close()
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.pongCH:
			err := c.conn.WriteMessage(2, msg[:])
			if err != nil {
				fmt.Printf("error sending msg %v\n", err)
				continue
			}

		case msg := <-c.msgCH:
			err := c.sendMsg(msg)
			if err != nil {
				return
			}
		}
	}
}

func (c *TestClient) sendMsg(msg interface{}) error {
	err := c.conn.WriteJSON(msg)
	if err != nil {
		fmt.Printf("error sending msg %v\n", err)
		return err
	}
	return nil
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

			msg := new(RespMsg)
			err = json.Unmarshal(b, msg)
			if err != nil {
				log.Fatal("unable to unmarshal json")
			}
			if msg.MsgType == MsgType_Ping {
				pongMsg := [1]byte{}
				c.pongCH <- pongMsg
			}

			// if msg.MsgType == MsgType_Throttled {
			// 	tc.throttledMsgCount.Add(1)
			// } else {
			// 	tc.brMsgCount.Add(1)
			// }
			tc.brMsgCount.Add(1)
			tc.lastOffset = msg.Offset
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

func TestBroadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer()
	go s.CreateWSServer()
	time.Sleep(1 * time.Second)
	clientCount := 5
	brCount := 2

	tc := TestConfig{
		clientCount:    clientCount,
		wg:             new(sync.WaitGroup),
		brMsgCount:     new(atomic.Int64),
		targetMsgCount: clientCount * brCount,
	}
	tc.wg.Add(tc.clientCount + 1)

	brConn := DialServer(&tc)
	brClient := NewTestClient(brConn, ctx)
	go brClient.writeLoop()

	for range tc.clientCount {
		go DialServer(&tc)
	}
	time.Sleep(1 * time.Second)

	for range brCount {
		msg := ReqMsg{
			MsgType: MsgType_Broadcast,
			Data:    "hello from tests",
		}
		brClient.msgCH <- &msg
	}

	tc.wg.Wait()
	cancel()

	time.Sleep(1 * time.Second)
	fmt.Println("exiting test")
}

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

func (c *TestClient) SendReplay(tc *TestConfig, roomID string, offset int) {
	msg := NewReqMsg(MsgType_Replay, roomID, fmt.Sprintf("wanna replay from offset = %d", offset))
	c.msgCH <- msg
}

func TestRooms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer()
	go s.CreateWSServer()
	time.Sleep(1 * time.Second)
	clientCount := 2
	msgCount := 2

	<-s.MsgConsumer.readyCH

	tc := TestConfig{
		clientCount:    clientCount,
		wg:             new(sync.WaitGroup),
		brMsgCount:     new(atomic.Int64),
		targetMsgCount: 0,
	}

	rID1 := "FIRST_ROOM"
	rID2Count := 0
	rID2 := "SECOND_ROOM"
	tc.wg.Add(tc.clientCount)
	clients := []*TestClient{}
	for idx := range tc.clientCount {
		conn := JoinServer(&tc)
		client := NewTestClient(conn, ctx)
		clients = append(clients, client)
		go client.readLoop(&tc)
		go client.writeLoop()
		rID := rID1
		if rand.IntN(10) < 5 || idx == 0 {
			rID = rID2
			rID2Count++
		}

		client.roomID = rID
		client.JoinRoom(&tc, rID)
	}

	room1Clients := tc.clientCount - rID2Count
	room2Clients := rID2Count

	expectedMessages := 0
	for idx := range msgCount {
		if clients[idx].roomID == rID1 {
			expectedMessages += room1Clients - 1
		} else {
			expectedMessages += room2Clients - 1
		}
	}
	tc.targetMsgCount = expectedMessages

	for {
		res := s.GetTestResult(rID1)
		fmt.Printf("test_res = %v\n", res)
		if res.clientsCount == tc.clientCount-rID2Count {
			break
		}
	}

	for idx := range msgCount {
		c := clients[idx]
		go c.SendRoomMsg(&tc, c.roomID)
	}

	for {
		res1 := s.GetTestResult(rID1)
		res2 := s.GetTestResult(rID2)
		currMsgCount := int(tc.brMsgCount.Load())
		fmt.Printf("test_res = %v, curr_msg_count = %d\n", res1, currMsgCount)
		if res1.clientsCount == room1Clients &&
			res2.clientsCount == room2Clients &&
			currMsgCount == tc.targetMsgCount {
			break
		}
		time.Sleep(1 * time.Second)
	}

	for _, client := range clients {
		client.LeaveRoom(&tc, client.roomID)
	}

	for {
		res1 := s.GetTestResult(rID1)
		res2 := s.GetTestResult(rID2)
		if res1.clientsCount == 0 && res2.clientsCount == 0 {
			for _, client := range clients {
				go func(conn *websocket.Conn) {
					conn.Close()
					tc.wg.Done()
				}(client.conn)
			}
			break
		}
		time.Sleep(1 * time.Second)
	}

	tc.wg.Wait()
	cancel()

	fmt.Println("exiting test")
}

func TestBackPressure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer()
	go s.CreateWSServer()
	time.Sleep(1 * time.Second)
	clientCount := 2
	brCount := 30

	tc := TestConfig{
		clientCount:    clientCount,
		wg:             new(sync.WaitGroup),
		brMsgCount:     new(atomic.Int64),
		targetMsgCount: (clientCount - 1) * brCount,
	}

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
	}

	msg := NewReqMsg(MsgType_Broadcast, "", "wanna send broadcast")

	for range brCount {
		clients[0].msgCH <- msg
	}

	for {
		time.Sleep(time.Second)
		dropped := s.droppedMsgCount.Load()
		fmt.Printf("receivedCount = %d, target = %d, dropped = %d\n", tc.brMsgCount.Load(), tc.targetMsgCount, dropped)
		if int(tc.brMsgCount.Load())+int(dropped) == tc.targetMsgCount {
			break
		}
	}

	cancel()

	fmt.Println("exiting test")
}

func TestThrottling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer()
	go s.CreateWSServer()
	time.Sleep(1 * time.Second)
	clientCount := 5
	brCount := 20

	tc := TestConfig{
		clientCount:       clientCount,
		wg:                new(sync.WaitGroup),
		brMsgCount:        new(atomic.Int64),
		targetMsgCount:    (clientCount - 1) * brCount,
		throttledMsgCount: new(atomic.Int64),
	}

	clients := []*TestClient{}

	for range tc.clientCount {
		conn := JoinServer(&tc)
		client := NewTestClient(conn, ctx)
		clients = append(clients, client)
		go client.readLoop(&tc)
		go client.writeLoop()
	}

	time.Sleep(1 * time.Second)
	timeStart := time.Now()
	msg := NewReqMsg(MsgType_Broadcast, "", "wanna send broadcast")

	for range brCount {
		clients[0].msgCH <- msg
	}

	for {
		time.Sleep(time.Second)
		fmt.Printf("receivedCount = %d, target = %d\n", tc.brMsgCount.Load(), tc.targetMsgCount)
		// dropped := s.droppedMsgCount.Load()
		// if int(tc.brMsgCount.Load())+int(dropped)+int(tc.throttledMsgCount.Load()) == tc.targetMsgCount {
		// 	break
		// }
		if int(tc.brMsgCount.Load()) == tc.targetMsgCount {
			break
		}

	}

	testDuration := time.Since(timeStart)

	// 1. Sender's rate (this is what's throttled)
	messagesSent := float64(brCount) // 50 messages sent by client[0]
	senderRate := messagesSent / testDuration.Seconds()

	// 2. Total throughput (all clients receiving)
	messagesReceived := float64(tc.brMsgCount.Load()) // 200 messages received by 4 clients
	totalThroughput := messagesReceived / testDuration.Seconds()

	// 3. Expected values
	expectedSenderRate := float64(ThrottlerMessagesPerSecond)              // 3 msg/sec
	expectedTotalThroughput := expectedSenderRate * float64(clientCount-1) // 3 * 4 = 12 msg/sec

	// 4. Expected minimum duration
	expectedMinDuration := time.Duration(float64(brCount)/float64(ThrottlerMessagesPerSecond)) * time.Second

	// Print results
	fmt.Printf("\n=== TEST RESULTS ===\n")
	fmt.Printf("Test duration: %v\n", testDuration)
	fmt.Printf("Expected min duration: %v\n", expectedMinDuration)
	fmt.Printf("\n")
	fmt.Printf("Messages sent (by client[0]): %d\n", brCount)
	fmt.Printf("Messages received (by 4 clients): %d\n", int(messagesReceived))
	fmt.Printf("\n")
	fmt.Printf("SENDER RATE (what's throttled):\n")
	fmt.Printf("  Actual: %.2f msg/sec\n", senderRate)
	fmt.Printf("  Expected: %.2f msg/sec\n", expectedSenderRate)
	fmt.Printf("\n")
	fmt.Printf("TOTAL THROUGHPUT (all receivers):\n")
	fmt.Printf("  Actual: %.2f msg/sec\n", totalThroughput)
	fmt.Printf("  Expected: %.2f msg/sec (sender rate × num receivers)\n", expectedTotalThroughput)
	fmt.Printf("\n")
	fmt.Printf("Throttled responses: %d\n", tc.throttledMsgCount.Load())
	fmt.Printf("\n=== END RESULTS ===\n")

	cancel()

	// CORRECTED ASSERTIONS

	// Assert 1: Test should take long enough
	assert.GreaterOrEqual(t, testDuration, expectedMinDuration*9/10,
		"Test should take at least the minimum time for throttling")

	// Assert 2: Sender rate should be close to throttle limit
	assert.InEpsilon(t, expectedSenderRate, senderRate, 0.3,
		"Sender rate should be close to throttle limit (3 msg/sec)")

	// Assert 3: Total throughput should be sender_rate × num_receivers
	assert.InEpsilon(t, expectedTotalThroughput, totalThroughput, 0.3,
		"Total throughput should equal sender rate × number of receivers")

	// Optional: Check if any messages were throttled
	if tc.throttledMsgCount.Load() > 0 {
		fmt.Printf("\n✓ Some messages were throttled (rate limiting kicked in)\n")
	}

	for {
		time.Sleep(time.Second)
		if len(s.clients) == 0 {
			break
		}
	}

	fmt.Println("\nexiting test")
}
