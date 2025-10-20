package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type MsgType string

const (
	MsgType_JoinRoom     MsgType = "join-room"
	MsgType_LeaveRoom    MsgType = "leave-room"
	MsgType_SendRoomMsg  MsgType = "room-message"
	MsgType_BroadcastMsg MsgType = "broadcast"
)

type Message struct {
	RoomID  string      `json:"roomID"`
	Data    interface{} `json:"data"`
	MsgType MsgType     `json:"type"`
	Client  *Client
}

func NewMessage(c *Client) *Message {
	return &Message{
		Client: c,
	}
}

type Response struct {
	RoomID   string      `json:"roomID"`
	Data     interface{} `json:"data"`
	MsgType  MsgType     `json:"type"`
	SenderID string      `json:"senderID"`
}

func NewResponse(msg *Message) *Response {
	return &Response{
		RoomID:   msg.RoomID,
		Data:     msg.Data,
		MsgType:  msg.MsgType,
		SenderID: msg.Client.id,
	}
}

type Client struct {
	id    string
	rooms map[string]*Room
	mu    *sync.RWMutex
	conn  *websocket.Conn
}

func NewClient(conn *websocket.Conn) *Client {
	id := rand.Text()[:9]
	return &Client{
		id:    id,
		conn:  conn,
		rooms: map[string]*Room{},
		mu:    &sync.RWMutex{},
	}
}

func (c *Client) readMsgLoop(ctx context.Context, srv *WSServer) {
	defer func() {
		c.conn.Close()
		srv.leaveServerCH <- c
	}()

	go func() {
		<-ctx.Done()
		c.conn.Close()
	}()

	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			// TODO add test .env to enable/disable
			// logrus.Errorf("error reading msg loop for client = %s, err = %v", c.id, err)
			return
		}

		msg := NewMessage(c)
		if len(b) == 1 {
			// TODO -> ping/pong
			logrus.WithField("clientID", c.id).Info("received pong")
			continue
		}

		err = json.Unmarshal(b, &msg)
		if err != nil {
			logrus.Error("error unmarshaling msg", err)
			continue
		}

		if msg.MsgType == "" && msg.RoomID != "" {
			logrus.Error("invalid msg format, msgType is required ", err)
			continue
		}
		msg.Client = c
		if msg.RoomID != "" {
			switch msg.MsgType {
			case MsgType_JoinRoom:
				srv.joinRoomCH <- msg
				continue
			case MsgType_LeaveRoom:
				srv.leaveRoomCH <- msg
				continue
			case MsgType_SendRoomMsg:
				srv.sendRoomMsgCH <- msg
				continue
			}
		} else {
			srv.sendBroadcastMsgCH <- msg
		}
	}
}

type Room struct {
	id      string
	clients map[string]*Client
	mu      *sync.RWMutex
}

func NewRoom(id string) *Room {
	return &Room{
		id:      id,
		clients: map[string]*Client{},
		mu:      &sync.RWMutex{},
	}
}

type WSServer struct {
	clients map[string]*Client
	rooms   map[string]*Room
	mu      *sync.RWMutex

	leaveServerCH      chan *Client
	joinServerCH       chan *Client
	sendBroadcastMsgCH chan *Message
	joinRoomCH         chan *Message
	leaveRoomCH        chan *Message
	sendRoomMsgCH      chan *Message
	errCh              chan error
	shutdownCH         chan struct{}

	ctx      context.Context
	cancelFN context.CancelFunc
	wg       *sync.WaitGroup

	// for testing
	activeClients atomic.Int64
}

func NewWSServer(ctx context.Context, cancelFN context.CancelFunc) *WSServer {
	return &WSServer{
		clients: map[string]*Client{},
		rooms:   map[string]*Room{},
		mu:      &sync.RWMutex{},

		leaveServerCH:      make(chan *Client, 64),
		joinServerCH:       make(chan *Client, 64),
		sendBroadcastMsgCH: make(chan *Message, 64),
		joinRoomCH:         make(chan *Message, 64),
		leaveRoomCH:        make(chan *Message, 64),
		sendRoomMsgCH:      make(chan *Message, 64),
		errCh:              make(chan error, 64),
		shutdownCH:         make(chan struct{}),

		ctx:      ctx,
		cancelFN: cancelFN,
		wg:       new(sync.WaitGroup),
	}
}

// for testing
func (srv *WSServer) GetClientCount() int {
	return int(srv.activeClients.Load())
}

func (srv *WSServer) wsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  512,
			WriteBufferSize: 512,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logrus.Error("error upgrading ws conn")
		}

		c := NewClient(conn)
		go func() {
			srv.joinServerCH <- c
		}()
		go c.readMsgLoop(srv.ctx, srv)
	}
}

func (srv *WSServer) AcceptLoop() {
	for {
		select {
		case <-srv.ctx.Done():
			fmt.Println("exiting accept loop")
			go srv.ShutdownLoopAtomic()
			return
		case err := <-srv.errCh:
			logrus.Error(err)
		case client := <-srv.leaveServerCH:
			srv.LeaveServer(client)
		case client := <-srv.joinServerCH:
			srv.JoinServer(client)
		case msg := <-srv.joinRoomCH:
			srv.JoinRoom(msg)
		case msg := <-srv.leaveRoomCH:
			srv.LeaveRoom(msg)
		case msg := <-srv.sendRoomMsgCH:
			go srv.SendRoomMsg(msg)
		case msg := <-srv.sendBroadcastMsgCH:
			go srv.SendBroadcastMsg(msg)
		}
	}
}

func (srv *WSServer) ShutdownLoop() {
	defer close(srv.shutdownCH)
	fmt.Println("starting shutdown")
	// timeout := time.After(60 * time.Second)
	closeCH := make(chan struct{})

	go func() {
		srv.wg.Wait()
		close(closeCH)
		fmt.Println("exiting shutdown")
	}()

	for {
		select {
		case <-closeCH:
			return
		// case <-timeout:
		// 	fmt.Println("EXIT DUE TO TIMEOUT")
		// return
		case client := <-srv.leaveServerCH:
			srv.LeaveServer(client)
		}
	}
}

func (srv *WSServer) ShutdownLoopAtomic() {
	defer close(srv.shutdownCH)
	fmt.Println("starting shutdown")
	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-timeout:
			fmt.Println("EXIT DUE TO TIMEOUT")
			return
		case client := <-srv.leaveServerCH:
			srv.LeaveServer(client)
			if srv.activeClients.Load() == 0 {
				return
			}
			// default:
			// 	if srv.activeClients.Load() == 0 {
			// 		return
			// 	}
		}
	}
}

func (srv *WSServer) LeaveServer(client *Client) {
	_, ok := srv.clients[client.id]
	if ok {
		// defer srv.wg.Done()
		srv.activeClients.Add(-1)
		delete(srv.clients, client.id)
	}

	for _, r := range client.rooms {
		delete(r.clients, client.id)
	}
	client.rooms = nil
	// logrus.WithField("id", client.id).Info("client left server")
	fmt.Printf("client count AFTER exit = %d\n", srv.activeClients.Load())
}

func (srv *WSServer) JoinServer(client *Client) {
	_, ok := srv.clients[client.id]
	if ok {
		logrus.WithField("id", client.id).Info("cleint already exists")
		return
	}

	// srv.wg.Add(1)
	srv.clients[client.id] = client
	srv.activeClients.Add(1)
	logrus.WithField("id", client.id).Info("client joined server")
}

func (srv *WSServer) JoinRoom(msg *Message) {
	srv.JoinServer(msg.Client)
	room, ok := srv.rooms[msg.RoomID]

	if !ok {
		room = NewRoom(msg.RoomID)
		srv.rooms[msg.RoomID] = room
	}

	room.clients[msg.Client.id] = msg.Client
	msg.Client.rooms[room.id] = room

	logrus.WithFields(
		logrus.Fields{
			"clientID":     msg.Client.id,
			"roomID":       msg.RoomID,
			"clientsCount": len(room.clients),
		},
	).Info("client joined room")

}

func (srv *WSServer) LeaveRoom(msg *Message) {
	room, ok := srv.rooms[msg.RoomID]
	if !ok {
		logrus.WithFields(
			logrus.Fields{
				"clientID": msg.Client.id,
				"roomID":   msg.RoomID,
			},
		).Error("client cannot leave non existing room")
		return
	}

	delete(room.clients, msg.Client.id)
	delete(msg.Client.rooms, msg.RoomID)
	logrus.WithFields(
		logrus.Fields{
			"clientID":     msg.Client.id,
			"roomID":       msg.RoomID,
			"clientsCount": len(room.clients),
		},
	).Info("client left room")
}

func (srv *WSServer) SendRoomMsg(msg *Message) {
	room, ok := srv.rooms[msg.RoomID]
	if !ok {
		logrus.WithFields(
			logrus.Fields{
				"roomID": msg.RoomID,
			},
		).Error("cannot send to non existing room")
		return
	}

	cls := []*Client{}
	room.mu.RLock()
	for _, c := range room.clients {
		if c.id == msg.Client.id {
			continue
		}
		cls = append(cls, c)
	}
	room.mu.RUnlock()

	for _, c := range cls {
		resp := NewResponse(msg)
		err := c.conn.WriteJSON(resp)
		if err != nil {
			logrus.Error("Error sending msg", err)
			if websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				go func() {
					srv.leaveServerCH <- msg.Client
				}()
			}
		}
	}
	logrus.WithFields(
		logrus.Fields{
			"senderID":      msg.Client.id,
			"roomID":        msg.RoomID,
			"receiverCount": len(cls),
		},
	).Info("sent room-msg")

}

func (srv *WSServer) SendBroadcastMsg(msg *Message) {
	cls := []*Client{}

	srv.mu.RLock()
	for _, c := range srv.clients {
		if c.id == msg.Client.id {
			continue
		}
		cls = append(cls, c)
	}
	srv.mu.RUnlock()

	msg.RoomID = ""
	for _, c := range cls {
		resp := NewResponse(msg)
		err := c.conn.WriteJSON(resp)
		if err != nil {
			logrus.Error("Error sending msg", err)
			if websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				go func() {
					srv.leaveServerCH <- msg.Client
				}()
			}
		}
	}

	logrus.WithFields(
		logrus.Fields{
			"senderID":      msg.Client.id,
			"receiverCount": len(cls),
		},
	).Info("sent broadcast")
}

func (srv *WSServer) cleanUp() {
	close(srv.leaveServerCH)
	close(srv.joinServerCH)
	close(srv.sendBroadcastMsgCH)
	close(srv.joinRoomCH)
	close(srv.leaveRoomCH)
	close(srv.sendRoomMsgCH)
	close(srv.errCh)

	// timeout := time.After(20 * time.Second)
	// for {
	// 	select {
	// 	case client := <-srv.leaveServerCH:
	// 		srv.LeaveServer(client)
	// 		if srv.activeClients.Load() == 0 {
	// 			return
	// 		}
	// 	case <-timeout:
	// 		logrus.Warn("exiting due to timeout exceeded")
	// 		return
	// 	default:
	// 		if srv.activeClients.Load() == 0 {
	// 			return
	// 		}
	// 	}
	// }
}

// read about logrus perf-ce
// read about sync.Map vs Map + mu.Lock
// research queue to remove mu.Lock
func chatServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wsSrv := NewWSServer(ctx, cancel)
	http.HandleFunc("/", wsSrv.wsHandler())
	go wsSrv.AcceptLoop()

	logrus.Fatal(http.ListenAndServe(":3231", nil))
}
