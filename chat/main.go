package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var CloseSignalToHandle = []int{
	websocket.CloseNormalClosure,
	websocket.CloseGoingAway,
	websocket.CloseProtocolError,
	websocket.CloseUnsupportedData,
	websocket.CloseNoStatusReceived,
	websocket.CloseAbnormalClosure,
	websocket.CloseInvalidFramePayloadData,
	websocket.ClosePolicyViolation,
	websocket.CloseMessageTooBig,
	websocket.CloseMandatoryExtension,
	websocket.CloseInternalServerErr,
	websocket.CloseServiceRestart,
	websocket.CloseTryAgainLater,
	websocket.CloseTLSHandshake,
}

type MsgType string

const (
	// TODO -> move somewhere else?
	MsgType_JoinRoom  MsgType = "join-room"
	MsgType_LeaveRoom MsgType = "leave-room"

	MsgType_RoomMessage MsgType = "room-message"
	MsgType_Broadcast   MsgType = "broadcast"
	MsgType_Direct      MsgType = "direct"
	// TODO -> change min possible msg size, 1byte?
	MsgType_Ping MsgType = "ping"
	MsgType_Pong MsgType = "pong"
)

type RegisterCmd string

const (
	RegisterCmd_RegisterBroadcast RegisterCmd = "register-broadcast"
	RegisterCmd_RegisterRoom      RegisterCmd = "register-room"
	RegisterCmd_UnregisterRoom    RegisterCmd = "unregister-room"
	RegisterCmd_Unregister        RegisterCmd = "unregister"
)

type Client struct {
	conn      *websocket.Conn
	msgChan   chan *ClientMSG
	wsID      string
	rooms     map[string]*Room
	mu        *sync.Mutex
	pingCount int
	closeCH   chan struct{}
}

func NewClient(conn *websocket.Conn) *Client {
	wsID := rand.Text()
	logrus.Infof("new client conn %s", wsID)
	return &Client{
		conn:    conn,
		msgChan: make(chan *ClientMSG, 64),
		closeCH: make(chan struct{}),
		wsID:    wsID,
		rooms:   make(map[string]*Room),
		mu:      new(sync.Mutex),
	}
}

type Room struct {
	ID      string
	Clients map[string]*Client
	mu      *sync.RWMutex
}

func NewRoom(id string) *Room {
	return &Room{
		ID:      id,
		mu:      new(sync.RWMutex),
		Clients: make(map[string]*Client),
	}
}

type RegisterPayload struct {
	client *Client
	cmd    RegisterCmd
	roomID string
}

type WSServer struct {
	Clients    map[string]*Client
	Rooms      map[string]*Room
	mu         *sync.RWMutex
	registerCH chan *RegisterPayload
}

func NewWSServer() *WSServer {
	return &WSServer{
		Clients:    make(map[string]*Client),
		Rooms:      make(map[string]*Room),
		mu:         new(sync.RWMutex),
		registerCH: make(chan *RegisterPayload, 16),
	}
}

func (svr *WSServer) RegisterBroadcast(client *Client) {
	svr.mu.Lock()
	svr.Clients[client.wsID] = client
	svr.mu.Unlock()
	logrus.WithFields(logrus.Fields{
		"clientsCount": len(svr.Clients),
		"newClientID":  client.wsID,
	}).Info("client joined broadcast")

}

func (svr *WSServer) Unregister(c *Client) {
	logrus.WithFields(logrus.Fields{
		"step": "unregister from server",
		"wsID": c.wsID,
	}).Infof("received close signal")

	svr.mu.Lock()
	_, ok := svr.Clients[c.wsID]
	if !ok {
		logrus.Warnf("wsID %s is not registered, no one to unregister", c.wsID)
		return
	}
	delete(svr.Clients, c.wsID)
	svr.mu.Unlock()

	c.mu.Lock()
	for _, r := range c.rooms {
		delete(r.Clients, c.wsID)
	}
	c.mu.Unlock()

	logrus.Infof("client removed from server %s", c.wsID)
}

func (svr *WSServer) RegisterLoop() {
	defer close(svr.registerCH)
	for {
		v := <-svr.registerCH
		switch v.cmd {
		case RegisterCmd_RegisterBroadcast:
			svr.RegisterBroadcast(v.client)
		case RegisterCmd_RegisterRoom:
			svr.JoinRoom(v.client, v.roomID)
		case RegisterCmd_Unregister:
			svr.Unregister(v.client)
		case RegisterCmd_UnregisterRoom:
			svr.LeaveRoom(v.client, v.roomID)
		}
	}
}

func (svr *WSServer) BroadcastMSG(msg interface{}) {
	logrus.WithFields(logrus.Fields{
		"ClientsCount": len(svr.Clients),
		"msg":          msg,
	}).Infof("broadcasting msg to all clients")

	for _, c := range svr.Clients {
		err := c.conn.WriteJSON(msg)
		if err != nil {
			logrus.Errorf("error sending msg to client %s, %v", c.wsID, err)
			continue
		}
		logrus.WithField("wsID", c.wsID).Info("send msg success")
	}
}

func (svr *WSServer) SendRoomMSG(msg interface{}, clientID string, roomID string) error {
	svr.mu.RLock()
	room, ok := svr.Rooms[roomID]
	if !ok {
		errMsg := fmt.Sprintf("roomID %s does not exist, cannot send msg", roomID)
		logrus.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	svr.mu.RUnlock()

	if room == nil {
		return fmt.Errorf("roomID does not exist %s", roomID)
	}

	validClient := false
	clientsToNotify := []*Client{}
	for _, v := range room.Clients {
		if v.wsID == clientID {
			validClient = true
		} else {
			clientsToNotify = append(clientsToNotify, v)
		}
	}
	if !validClient {
		return fmt.Errorf("cliendID: %s does not belong to this roomID: %s", clientID, roomID)
	}

	for _, c := range clientsToNotify {
		err := c.conn.WriteJSON(msg)
		if err != nil {
			logrus.Errorf("error sending msg to client %s, %v", c.wsID, err)
			continue
		}
		logrus.WithField("wsID", c.wsID).Info("send msg success")
	}

	return nil
}

func (svr *WSServer) JoinRoom(client *Client, roomID string) {
	svr.mu.Lock()
	room, ok := svr.Rooms[roomID]

	if !ok {
		room = NewRoom(roomID)
		svr.Rooms[roomID] = room
	}
	svr.mu.Unlock()

	room.mu.Lock()
	room.Clients[client.wsID] = client
	room.mu.Unlock()

	client.mu.Lock()
	client.rooms[roomID] = room
	client.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"roomID":           roomID,
		"clientID":         client.wsID,
		"totalRooms":       len(svr.Rooms),
		"totalClients":     len(svr.Clients),
		"clientsInTheRoom": len(room.Clients),
	}).Info("client joined room")
}

func (svr *WSServer) LeaveRoom(client *Client, roomID string) {
	svr.mu.Lock()
	room, ok := svr.Rooms[roomID]

	if !ok {
		logrus.Warnf("Error leaving non-registered room -> %s does not exist", roomID)
	}
	delete(svr.Rooms, roomID)
	svr.mu.Unlock()

	if ok {
		room.mu.Lock()
		client, ok := room.Clients[client.wsID]
		if !ok {
			logrus.Warnf("client %s does not belong to the room %s we are trying to leave", client.wsID, roomID)
		}
		delete(room.Clients, client.wsID)
		room.mu.Unlock()
	}

	client.mu.Lock()
	delete(client.rooms, roomID)
	client.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"roomID":           roomID,
		"clientID":         client.wsID,
		"totalRooms":       len(svr.Rooms),
		"totalClients":     len(svr.Clients),
		"clientsInTheRoom": len(room.Clients),
	}).Info("client left room")
}

func (svr *WSServer) Ping(client *Client) {
	msg := &ClientMSG{
		MsgType: MsgType_Pong,
	}
	client.conn.WriteJSON(msg)
}

type ClientMSG struct {
	MsgType MsgType `json:"type"`
	// TODO -> how to make generic type here? or any is the right way?
	Data   interface{} `json:"data"`
	RoomID string      `json:"roomID"`
}

func (svr *WSServer) handleWS() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.RequestURI
		parsedCMD, err := parseURLToCmd(h)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		upgrader := websocket.Upgrader{
			// TODO-> what happens if buffer to small?
			ReadBufferSize:  512,
			WriteBufferSize: 512,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatalf("error updagrading WS %v", err)
		}

		client := NewClient(conn)

		go svr.processCMD(client, parsedCMD)
		go client.readMsgLoop(svr)
		time.Sleep(time.Millisecond * 50)
		go client.wsAcceptLoop()
	}

}

func (svr *WSServer) processCMD(client *Client, cmd *ParsedCMD) {
	switch cmd.msgType {
	case MsgType_JoinRoom:
		svr.registerCH <- &RegisterPayload{
			client: client,
			cmd:    RegisterCmd_RegisterRoom,
			roomID: cmd.roomID,
		}
		return
	case MsgType_LeaveRoom:
		svr.registerCH <- &RegisterPayload{
			client: client,
			cmd:    RegisterCmd_UnregisterRoom,
			roomID: cmd.roomID,
		}
		return
	case MsgType_Broadcast:
	default:
		svr.registerCH <- &RegisterPayload{
			client: client,
			cmd:    RegisterCmd_RegisterBroadcast,
			roomID: "",
		}
		return
	}

}

func (c *Client) wsAcceptLoop() {
	defer func() {
		logrus.Info("exiting wsAcceptLoop")
		close(c.msgChan)
	}()

	readCH := make(chan *ClientMSG, 64)
	readErrCH := make(chan error, 16)

	go func() {
		defer close(readCH)
		defer close(readErrCH)
		for {
			msg := new(ClientMSG)
			err := c.conn.ReadJSON(msg)
			if err != nil {
				logrus.Errorf("error reading conn %v", err)
				readErrCH <- err
				return
			}
			readCH <- msg
		}
	}()

	for {
		logrus.Infof("accept loop -> receiving")

		select {
		case <-readErrCH:
			logrus.Errorf("exiting wsAcceptLoop, due to err")
			return
		case msg := <-readCH:
			logrus.Infof("sending msg %v", msg)
			c.msgChan <- msg
		}
	}
}

func (c *Client) readMsgLoop(svr *WSServer) {
	t := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-t.C:
			logrus.Infof("current ping count = %d", c.pingCount)
			c.pingCount--
			if c.pingCount < -2 {
				err := c.conn.Close()
				if err != nil {
					logrus.Errorf("err closing connection %v", err)
				}
			}
		case v, ok := <-c.msgChan:
			if !ok {
				logrus.WithFields(logrus.Fields{
					"step": "exit readMsgLoop",
					"wsID": c.wsID,
				}).Infof("received close signal")
				svr.registerCH <- &RegisterPayload{
					client: c,
					cmd:    RegisterCmd_Unregister,
				}
				return
			}
			switch v.MsgType {
			case MsgType_Broadcast:
				svr.BroadcastMSG(v.Data)
			case MsgType_RoomMessage:
				svr.SendRoomMSG(v.Data, c.wsID, v.RoomID)
			case MsgType_JoinRoom:
				svr.JoinRoom(c, v.RoomID)
			case MsgType_LeaveRoom:
				svr.LeaveRoom(c, v.RoomID)
			case MsgType_Ping:
				c.pingCount++
				svr.Ping(c)
			}
			logrus.WithFields(logrus.Fields{
				"wsID":    c.wsID,
				"msgType": v.MsgType,
			}).Info("received WS msg")
		}
	}
}

type ParsedCMD struct {
	msgType MsgType
	roomID  string
}

func parseURLToCmd(val string) (*ParsedCMD, error) {
	res := strings.Split(val, "/")

	if len(res) > 1 {
		msgType := MsgType(res[1])
		switch msgType {
		case MsgType_Broadcast:
		case MsgType_Direct:
		case MsgType_RoomMessage:
			return &ParsedCMD{
				msgType: msgType,
			}, nil

		case MsgType_JoinRoom:
			roomID := res[2]
			if roomID == "" {
				return nil, fmt.Errorf("roomID required to join the room %s", val)
			}
			return &ParsedCMD{
				msgType: msgType,
				roomID:  roomID,
			}, nil
		case MsgType_LeaveRoom:
			roomID := res[2]
			if roomID == "" {
				return nil, fmt.Errorf("roomID required to leave the room %s", val)
			}
			return &ParsedCMD{
				msgType: msgType,
				roomID:  roomID,
			}, nil
		default:
			return &ParsedCMD{
				msgType: "",
			}, nil
		}
	}
	return nil, fmt.Errorf("invalid msgType, params combination %s", val)
}

// TODOs
// - add ping/pong
// - add different rooms
// - check socket.io -> what can be borrowed

func main() {
	svr := NewWSServer()
	go svr.RegisterLoop()

	http.HandleFunc("/", svr.handleWS())
	logrus.Info("starting to listen on localhost:3231")
	log.Fatal(http.ListenAndServe("localhost:3231", nil))
}
