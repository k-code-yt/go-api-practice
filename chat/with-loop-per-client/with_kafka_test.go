package withloopperclient

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRoomsWithKafka(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer()
	go s.CreateWSServer()
	time.Sleep(1 * time.Second)
	clientCount := 8
	msgCount := 4

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

	// for {
	// 	res1 := s.GetTestResult(rID1)
	// 	res2 := s.GetTestResult(rID2)
	// 	currMsgCount := int(tc.brMsgCount.Load())
	// 	fmt.Printf("test_res1 = %v, curr_msg_count1 = %d\n", res1, currMsgCount)
	// 	fmt.Printf("test_res2 = %v, curr_msg_count2 = %d\n", res2, currMsgCount)

	// 	c := clients[0]
	// 	go c.SendRoomMsg(&tc, c.roomID)
	// 	time.Sleep(1 * time.Second)
	// }

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
			fmt.Println("====TARGET_REACHED====")
			fmt.Printf("target = %d, curr_msg_count = %d\n", tc.targetMsgCount, currMsgCount)
			break
		}
		time.Sleep(1 * time.Second)
	}

	for _, client := range clients {
		go client.LeaveRoom(&tc, client.roomID)
	}

	for {
		res1 := s.GetTestResult(rID1)
		res2 := s.GetTestResult(rID2)
		if res1.clientsCount == 0 && res2.clientsCount == 0 {
			fmt.Println("====LEAVING_ROOMS_TARGET_REACHED====")
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
func TestInfiniteRoomsWithKafka(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer()
	go s.CreateWSServer()
	time.Sleep(1 * time.Second)
	clientCount := 4

	<-s.MsgConsumer.readyCH

	tc := TestConfig{
		clientCount:    clientCount,
		wg:             new(sync.WaitGroup),
		brMsgCount:     new(atomic.Int64),
		targetMsgCount: 0,
		lastOffset:     0,
	}

	rID1 := "FIRST_ROOM"
	rID2 := rID1
	rID2Count := 0
	// rID2 := "SECOND_ROOM"
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

	tc.targetMsgCount = 10

	for {
		time.Sleep(1 * time.Second)
		res := s.GetTestResult(rID1)
		fmt.Printf("test_res = %v\n", res)
		if res.clientsCount == tc.clientCount {
			fmt.Println("====CLIENT_COUNT_REACHED====")
			break
		}
	}

	c := clients[0]

	for {
		time.Sleep(1 * time.Second)

		res1 := s.GetTestResult(rID1)
		res2 := s.GetTestResult(rID2)
		currMsgCount := int(tc.brMsgCount.Load())
		fmt.Printf("test_res1 = %+v, curr_msg_count1 = %d\n", res1, currMsgCount)
		fmt.Printf("test_res2 = %+v, curr_msg_count2 = %d\n", res2, currMsgCount)

		if currMsgCount >= tc.targetMsgCount {
			fmt.Println("====MSG_TARGET_REACHED====")

			for _, msg := range res1.msgBuffer {
				fmt.Println("msgOffsets = ", msg.Offset)
			}

			c.LeaveRoom(&tc, rID1)
			time.Sleep(1 * time.Second)
			break
		}
		go c.SendRoomMsg(&tc, c.roomID)
	}

	c.JoinRoom(&tc, rID1)
	c.SendReplay(&tc, rID1, tc.lastOffset-5)

	time.Sleep(5 * time.Second)

	cancel()

	fmt.Println("exiting test")
}
