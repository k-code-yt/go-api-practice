package withloopperclient

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	WSPort       = ":3223"
	PingPongFreq = time.Second * 30
)

type MsgType string

const (
	MsgType_Broadcast MsgType = "broadcast"
	MsgType_RoomJoin  MsgType = "room-join"
	MsgType_RoomLeave MsgType = "room-leave"
	MsgType_RoomMsg   MsgType = "room-msg"
	MsgType_Ping      MsgType = "ping"
	MsgType_Throttled MsgType = "throttled"
)

type ReqMsg struct {
	MsgType MsgType `json:"type"`
	Client  *Client
	Data    interface{} `json:"data"`
	RoomID  string      `json:"roomID"`
}

func NewReqMsg(msgType MsgType, rID string, data string) *ReqMsg {
	return &ReqMsg{
		MsgType: msgType,
		Data:    data,
		RoomID:  rID,
	}
}

type RespMsg struct {
	MsgType  MsgType     `json:"type"`
	Data     interface{} `json:"data"`
	SenderID string      `json:"senderID"`
	RoomID   string      `json:"roomID"`
	ErrCode  int         `json:"errorCode"`
}

func NewRespMsg(msg *ReqMsg) *RespMsg {
	return &RespMsg{
		MsgType:  msg.MsgType,
		Data:     msg.Data,
		SenderID: msg.Client.ID,
	}
}

type Client struct {
	ID          string
	mu          *sync.RWMutex
	conn        *websocket.Conn
	msgCH       chan *RespMsg
	done        chan struct{}
	pongCounter *atomic.Int64
	// throttling
	throttler *Throttler

	// back-pressure
	bpStrategy      BPStrategy
	queueSize       *atomic.Int64
	droppedMsgCount *atomic.Int64
}

func NewClient(conn *websocket.Conn) *Client {
	ID := rand.Text()[:9]
	t := NewThrottler(ThrottlerMessagesPerSecond)
	return &Client{
		ID:          ID,
		mu:          new(sync.RWMutex),
		conn:        conn,
		msgCH:       make(chan *RespMsg, 64),
		done:        make(chan struct{}),
		pongCounter: new(atomic.Int64),
		// throttling
		throttler: t,

		// back-pressure
		queueSize:       new(atomic.Int64),
		droppedMsgCount: new(atomic.Int64),
		bpStrategy:      DefaultBackPressureStrategy,
	}

}

func (c *Client) writeMsgLoop() {
	t := time.NewTicker(PingPongFreq)

	defer c.conn.Close()
	defer t.Stop()

	for {
		if c.pongCounter.Load() > 2 {
			fmt.Printf("pong counter exceeded -> disconnecting cID= %s\n", c.ID)
			return
		}
		select {
		case <-t.C:
			c.pongCounter.Add(1)
			continue
		case <-c.done:
			return
		case msg := <-c.msgCH:
			c.queueSize.Add(-1)
			err := c.conn.WriteJSON(msg)
			if err != nil {
				fmt.Printf("error sending msg to clientID = %s\n", c.ID)
				return
			}
		}
	}
}

func (c *Client) readMsgLoop(s *Server) {
	defer func() {
		close(c.throttler.exit)
		close(c.done)
		s.leaveServerCH <- c
	}()

	go c.acceptThrottledMsgLoop(s.handleMsg)

	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		if len(b) == 1 {
			c.pongCounter.Add(-1)
			fmt.Printf("received pong from cID = %s, pongCounter = %d\n", c.ID, c.pongCounter.Load())
			continue
		}

		msg := new(ReqMsg)
		err = json.Unmarshal(b, msg)
		if err != nil {
			fmt.Printf("unable to unmarshal the msg %v\n", err)
			continue
		}
		msg.Client = c
		c.throttler.inputCH <- msg
		// isAllowed := c.throttler.Allow(msg)
		// if !isAllowed {
		// 	resp := &RespMsg{
		// 		MsgType:  MsgType_Throttled,
		// 		SenderID: c.ID,
		// 		ErrCode:  429,
		// 	}
		// 	c.msgCH <- resp
		// }
	}
}

func (c *Client) acceptThrottledMsgLoop(handler func(msg *ReqMsg)) {
	for {
		select {
		case <-c.done:
			fmt.Printf("leaving acceptThrottledMsgLoop on CLIENT_DONE, cID = %s\n", c.ID)
			return
		case msg, ok := <-c.throttler.outputCH:
			if !ok {
				fmt.Printf("leaving acceptThrottledMsgLoop on THROTTLE_EXIT, cID = %s\n", c.ID)
				return
			}
			handler(msg)
		}
	}
}

func (s *Server) handleMsg(msg *ReqMsg) {
	switch msg.MsgType {
	case MsgType_Broadcast:
		s.broadcastCH <- msg
	case MsgType_RoomJoin:
		s.roomJoinCH <- msg
	case MsgType_RoomLeave:
		s.roomLeaveCH <- msg
	case MsgType_RoomMsg:
		s.roomMsgCH <- msg
	default:
		fmt.Printf("unknown message type = %s\n", msg.MsgType)
	}

}

func (c *Client) initPing() {
	for {
		select {
		case <-c.done:
			return
		default:
			time.Sleep(PingPongFreq)
			pingMsg := RespMsg{
				MsgType: MsgType_Ping,
			}
			fmt.Printf("sending ping msg to cID= %s\n", c.ID)
			c.msgCH <- &pingMsg
		}
	}
}

type Room struct {
	ID      string
	clients map[string]*Client
	mu      *sync.RWMutex

	// for tests
	clientsCount *atomic.Int64
}

func NewRoom(ID string) *Room {
	return &Room{
		ID:      ID,
		clients: map[string]*Client{},
		mu:      new(sync.RWMutex),

		// for tests
		clientsCount: new(atomic.Int64),
	}
}

type Server struct {
	clients       map[string]*Client
	rooms         map[string]*Room
	mu            *sync.RWMutex
	joinServerCH  chan *Client
	leaveServerCH chan *Client
	broadcastCH   chan *ReqMsg
	roomJoinCH    chan *ReqMsg
	roomLeaveCH   chan *ReqMsg
	roomMsgCH     chan *ReqMsg

	// for tests
	roomsCount      *atomic.Int64
	testReq         chan string
	testResultCH    chan *TestResult
	droppedMsgCount *atomic.Int64
	droppedCH       chan struct{}
}

func NewServer() *Server {
	return &Server{
		clients:       map[string]*Client{},
		rooms:         map[string]*Room{},
		mu:            new(sync.RWMutex),
		joinServerCH:  make(chan *Client, 64),
		leaveServerCH: make(chan *Client, 64),
		broadcastCH:   make(chan *ReqMsg, 64),
		roomJoinCH:    make(chan *ReqMsg, 64),
		roomLeaveCH:   make(chan *ReqMsg, 64),
		roomMsgCH:     make(chan *ReqMsg, 64),

		// for tests
		roomsCount:      new(atomic.Int64),
		testReq:         make(chan string, 16),
		testResultCH:    make(chan *TestResult, 64),
		droppedMsgCount: new(atomic.Int64),
		droppedCH:       make(chan struct{}, 64),
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error on HTTP conn upgrade %v\n", err)
		return
	}

	client := NewClient(conn)
	s.joinServerCH <- client

	go client.writeMsgLoop()
	go client.readMsgLoop(s)
	go client.initPing()
}

func (s *Server) AcceptLoop() {
	for {
		select {
		case c := <-s.joinServerCH:
			s.joinServer(c)
		case c := <-s.leaveServerCH:
			s.leaveServer(c)
		case msg := <-s.roomJoinCH:
			s.joinRoom(msg)
		case msg := <-s.roomLeaveCH:
			s.leaveRoom(msg)
		case msg := <-s.broadcastCH:
			go s.sendBroadcastMsg(msg)
		case msg := <-s.roomMsgCH:
			go s.sendRoomMsg(msg)

			// for tests
		case roomID := <-s.testReq:
			r := s.rooms[roomID]
			res := &TestResult{
				roomID:       roomID,
				clientsCount: int(r.clientsCount.Load()),
			}
			s.testResultCH <- res
		}
	}
}

func (s *Server) joinServer(c *Client) {
	s.clients[c.ID] = c
	fmt.Printf("client joined the server, cID = %s\n", c.ID)
}

func (s *Server) leaveServer(c *Client) {
	delete(s.clients, c.ID)
	for _, r := range s.rooms {
		_, ok := r.clients[c.ID]
		if ok {
			delete(r.clients, c.ID)
		}
	}

	fmt.Printf("client left the server, cID = %s\n", c.ID)
}

func (s *Server) sendMsg(msg *ReqMsg, cls map[string]*Client) {
	resp := NewRespMsg(msg)
	for _, c := range cls {
		c.msgCH <- resp
	}

	m := msg.RoomID
	if msg.RoomID == "" {
		m = "BROADCAST"
	}
	fmt.Printf("msg was sent to rID= %s | by cID= %s | num_clients=%d\n", m, msg.Client.ID, len(cls))
}

func (s *Server) backpressureSendMsg(msg *ReqMsg, cls map[string]*Client) {
	resp := NewRespMsg(msg)
	for _, c := range cls {
		c.handleBackpressure(resp, s.droppedCH)
	}

	m := msg.RoomID
	if msg.RoomID == "" {
		m = "BROADCAST"
	}
	fmt.Printf("msg was sent to rID= %s | by cID= %s | num_clients=%d\n", m, msg.Client.ID, len(cls))
}

func (s *Server) sendBroadcastMsg(msg *ReqMsg) {
	cls := map[string]*Client{}
	for id, c := range s.clients {
		if id != msg.Client.ID {
			cls[id] = c
		}
	}

	// go s.backpressureSendMsg(msg, cls)
	go s.sendMsg(msg, cls)
}

func (s *Server) sendRoomMsg(msg *ReqMsg) {
	r, ok := s.rooms[msg.RoomID]
	if !ok {
		fmt.Printf("roomID = %s does not exist\n", msg.RoomID)
		return
	}
	cls := map[string]*Client{}
	for id, c := range r.clients {
		if id != msg.Client.ID {
			cls[id] = c
		}
	}
	go s.backpressureSendMsg(msg, cls)
	// go s.sendMsg(msg, cls)
}

func (s *Server) joinRoom(msg *ReqMsg) {
	cID := msg.Client.ID
	s.clients[cID] = msg.Client

	room, ok := s.rooms[msg.RoomID]
	if !ok {
		room = NewRoom(msg.RoomID)
		s.rooms[msg.RoomID] = room
		s.roomsCount.Add(1)
	}

	room.clients[cID] = msg.Client

	room.clientsCount.Add(1)
	fmt.Printf("clientID %s joined the room %s\n", cID, msg.RoomID)
}

func (s *Server) leaveRoom(msg *ReqMsg) {
	cID := msg.Client.ID
	room, ok := s.rooms[msg.RoomID]
	if ok {
		delete(room.clients, cID)
		room.clientsCount.Add(-1)
	}

	fmt.Printf("clientID %s left the room %s\n", cID, msg.RoomID)
}

// for tests
type TestResult struct {
	clientsCount int
	roomID       string
}

func (s *Server) GetTestResult(roomID string) *TestResult {
	s.testReq <- roomID
	res := <-s.testResultCH
	return res
}

func (s *Server) CreateWSServer() {
	go s.AcceptLoop()
	go func() {
		for range s.droppedCH {
			s.droppedMsgCount.Add(1)
		}
	}()
	http.HandleFunc("/", s.handleWS)

	fmt.Printf("starting server on port: %s\n", WSPort)
	log.Fatal(http.ListenAndServe(WSPort, nil))
}
