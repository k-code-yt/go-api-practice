package withloopperclient

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

var (
	WSPort = ":3223"
)

type MsgType string

const (
	MsgType_Broadcast MsgType = "broadcast"
	MsgType_RoomJoin  MsgType = "room-join"
	MsgType_RoomLeave MsgType = "room-leave"
	MsgType_RoomMsg   MsgType = "room-msg"
)

type ReqMsg struct {
	MsgType MsgType
	Client  *Client
	Data    string
	RoomID  string
}

func NewReqMsg(msgType MsgType, rID string, data string) *ReqMsg {
	return &ReqMsg{
		MsgType: msgType,
		Data:    data,
		RoomID:  rID,
	}
}

type RespMsg struct {
	MsgType  MsgType
	Data     string
	SenderID string
	RoomID   string
}

func NewRespMsg(msg *ReqMsg) *RespMsg {
	return &RespMsg{
		MsgType:  msg.MsgType,
		Data:     msg.Data,
		SenderID: msg.Client.ID,
	}
}

type Client struct {
	ID    string
	mu    *sync.RWMutex
	conn  *websocket.Conn
	msgCH chan *RespMsg
	done  chan struct{}
}

func NewClient(conn *websocket.Conn) *Client {
	ID := rand.Text()[:9]
	return &Client{
		ID:    ID,
		mu:    new(sync.RWMutex),
		conn:  conn,
		msgCH: make(chan *RespMsg, 64),
		done:  make(chan struct{}),
	}

}

func (c *Client) writeMsgLoop() {
	defer c.conn.Close()
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.msgCH:
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
		close(c.done)
		s.leaveServerCH <- c
	}()

	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		msg := new(ReqMsg)
		err = json.Unmarshal(b, msg)
		if err != nil {
			fmt.Printf("unable to unmarshal the msg %v\n", err)
			continue
		}
		msg.Client = c

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
			continue
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
	roomsCount   *atomic.Int64
	testReq      chan string
	testResultCH chan *TestResult
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
		roomsCount:   new(atomic.Int64),
		testReq:      make(chan string, 16),
		testResultCH: make(chan *TestResult, 64),
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
			cls := map[string]*Client{}
			for id, c := range s.clients {
				if id != msg.Client.ID {
					cls[id] = c
				}
			}
			go s.sendMsg(msg, cls)
		case msg := <-s.roomMsgCH:
			r, ok := s.rooms[msg.RoomID]
			if !ok {
				fmt.Printf("roomID = %s does not exist\n", msg.RoomID)
			}
			cls := map[string]*Client{}
			for id, c := range r.clients {
				if id != msg.Client.ID {
					cls[id] = c
				}
			}
			go s.sendMsg(msg, cls)

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
	http.HandleFunc("/", s.handleWS)

	fmt.Printf("starting server on port: %s\n", WSPort)
	log.Fatal(http.ListenAndServe(WSPort, nil))
}
