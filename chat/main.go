package main

import (
	"crypto/rand"
	"encoding/json"
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
)

type RegisterCmd string

const (
	RegisterCmd_RegisterBroadcast RegisterCmd = "register-broadcast"
	RegisterCmd_RegisterRoom      RegisterCmd = "register-room"
	RegisterCmd_UnregisterRoom    RegisterCmd = "unregister-room"
	RegisterCmd_Unregister        RegisterCmd = "unregister"
)

type Client struct {
	conn    *websocket.Conn
	msgChan chan *ClientMSG
	wsID    string
	roomIds []string
	mu      *sync.Mutex
}

func NewClient(conn *websocket.Conn) *Client {
	wsID := rand.Text()
	logrus.Infof("new client conn %s", wsID)
	return &Client{
		conn:    conn,
		msgChan: make(chan *ClientMSG),
		wsID:    wsID,
		mu:      new(sync.Mutex),
	}
}

type Room struct {
	ID      string
	Clients []*Client
	mu      *sync.RWMutex
}

func NewRoom(id string) *Room {
	return &Room{ID: id, mu: new(sync.RWMutex)}
}

type RegisterPayload struct {
	client *Client
	cmd    RegisterCmd
	roomID string
}

type WSServer struct {
	Clients    []*Client
	Rooms      []*Room
	mu         *sync.RWMutex
	registerCH chan *RegisterPayload
}

func NewWSServer() *WSServer {
	return &WSServer{
		Clients:    []*Client{},
		Rooms:      []*Room{},
		mu:         new(sync.RWMutex),
		registerCH: make(chan *RegisterPayload, 24),
	}
}

func (svr *WSServer) RegisterBroadcast(client *Client) {
	svr.mu.Lock()
	svr.Clients = append(svr.Clients, client)
	svr.mu.Unlock()
	logrus.WithFields(logrus.Fields{
		"clientsCount": len(svr.Clients),
		"newClientID":  client.wsID,
	}).Info("client joined broadcast")

}

func (svr *WSServer) Unregister(client *Client) {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	var clientsCopy []*Client
	for _, v := range svr.Clients {
		if v.wsID == client.wsID {
			continue
		}
		clientsCopy = append(clientsCopy, v)
	}

	svr.Clients = nil
	svr.Clients = clientsCopy
	logrus.Infof("client removed from broadcast %s", client.wsID)
}

func (svr *WSServer) RegisterLoop() {
	defer close(svr.registerCH)
	for {
		v := <-svr.registerCH
		switch v.cmd {
		case RegisterCmd_RegisterBroadcast:
			svr.RegisterBroadcast(v.client)
		case RegisterCmd_RegisterRoom:
			svr.RegisterBroadcast(v.client)
			svr.AddToRoom(v.client, v.roomID)
		case RegisterCmd_Unregister:
			svr.Unregister(v.client)
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
	var room *Room
	for _, r := range svr.Rooms {
		if r.ID == roomID {
			room = r
			break
		}
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

func (svr *WSServer) AddToRoom(client *Client, roomID string) {
	svr.mu.RLock()
	var room *Room
	for _, r := range svr.Rooms {
		if r.ID == roomID {
			room = r
			break
		}
	}
	svr.mu.RUnlock()

	if room == nil {
		room = NewRoom(roomID)
		svr.mu.Lock()
		svr.Rooms = append(svr.Rooms, room)
		svr.mu.Unlock()
	}

	room.mu.Lock()
	room.Clients = append(room.Clients, client)
	room.mu.Unlock()

	client.mu.Lock()
	client.roomIds = append(client.roomIds, roomID)
	client.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"roomID":       roomID,
		"clientID":     client.wsID,
		"totalRooms":   len(svr.Rooms),
		"totalClients": len(svr.Clients),
	}).Info("client joined room")
}

type ClientMSG struct {
	MsgType MsgType     `json:"type"`
	Data    interface{} `json:"data"`
	RoomID  string      `json:"roomID"`
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
			ReadBufferSize:  56,
			WriteBufferSize: 56,
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
		go client.wsAcceptLoop(svr.registerCH)
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

func (c *Client) wsAcceptLoop(discChan chan *RegisterPayload) {
	defer close(c.msgChan)

	for {
		msg := new(ClientMSG)
		err := c.conn.ReadJSON(msg)
		if err != nil {
			if websocket.IsCloseError(err, CloseSignalToHandle...) {
				go func() {
					discChan <- &RegisterPayload{
						client: c,
						cmd:    RegisterCmd_Unregister,
					}
				}()
			}
			// TODO -> add leave room logic
			logrus.Infof("error reading conn %v", err)
			break
		}
		c.msgChan <- msg
	}
}

func (c *Client) readMsgLoop(svr *WSServer) {
	for v := range c.msgChan {
		switch v.MsgType {
		case MsgType_Broadcast:
			svr.BroadcastMSG(v.Data)
		case MsgType_RoomMessage:
			svr.SendRoomMSG(v.Data, c.wsID, v.RoomID)
		}

		// TODO -> add join room logic, based on room
		logrus.Info(v)
	}
}

func isValidJSON(data interface{}) bool {
	strVal, ok := data.(string)
	if !ok {
		return false
	}
	return json.Valid([]byte(strVal))
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

func main() {
	svr := NewWSServer()
	go svr.RegisterLoop()

	http.HandleFunc("/", svr.handleWS())
	logrus.Info("starting to listen on localhost:3231")
	log.Fatal(http.ListenAndServe("localhost:3231", nil))
}
