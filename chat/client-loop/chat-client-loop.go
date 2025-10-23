package otherclient

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type MsgType string

const (
	MsgType_JoinRoom    MsgType = "join-room"
	MsgType_LeaveRoom   MsgType = "leave-room"
	MsgType_RoomMessage MsgType = "room-message"
	MsgType_Broadcast   MsgType = "broadcast"
	MsgType_Direct      MsgType = "direct"
	MsgType_Ping        MsgType = "ping"
	MsgType_Pong        MsgType = "pong"
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
	msgChan   chan []byte
	wsID      string
	rooms     map[string]*Room
	mu        *sync.Mutex
	pingCount atomic.Int32
}

func NewClient(conn *websocket.Conn) *Client {
	wsID := rand.Text()
	logrus.Infof("new client conn %s", wsID)
	return &Client{
		conn:    conn,
		msgChan: make(chan []byte, 64),
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
	Clients          map[string]*Client
	Rooms            map[string]*Room
	registerCH       chan *RegisterPayload
	clientCH         chan *Client
	BroadcastWorkers []*BroadcastWorker

	mu *sync.RWMutex
	wg *sync.WaitGroup

	ctx    context.Context
	Cancel context.CancelFunc
}

func NewWSServer() *WSServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &WSServer{
		Clients:    make(map[string]*Client),
		Rooms:      make(map[string]*Room),
		registerCH: make(chan *RegisterPayload, 16),
		clientCH:   make(chan *Client, 16),

		mu: new(sync.RWMutex),
		wg: new(sync.WaitGroup),

		ctx:    ctx,
		Cancel: cancel,
	}
}

func (svr *WSServer) RegisterBroadcast(client *Client) {
	svr.mu.Lock()
	svr.Clients[client.wsID] = client
	svr.mu.Unlock()

	svr.clientCH <- client
	// TODO -> wait for close signal and close clientCH
	// i.e. cleanup

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
		svr.mu.Unlock()
		return
	}
	delete(svr.Clients, c.wsID)
	svr.mu.Unlock()

	c.mu.Lock()
	roomsToLeave := make([]*Room, 0, len(c.rooms))
	for _, r := range c.rooms {
		roomsToLeave = append(roomsToLeave, r)
	}
	c.mu.Unlock()

	for _, r := range roomsToLeave {
		r.mu.Lock()
		delete(r.Clients, c.wsID)
		r.mu.Unlock()
	}

	c.mu.Lock()
	c.rooms = make(map[string]*Room)
	c.mu.Unlock()

	logrus.Infof("client removed from server %s", c.wsID)
}

func (svr *WSServer) RegisterLoop() {
	defer close(svr.registerCH)

	for {
		select {
		case <-svr.ctx.Done():
			logrus.Info("received server close -> exiting RegisterLoop")
			return
		case v := <-svr.registerCH:
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
}

func (svr *WSServer) initUnregisterOnError(err error, c *Client) {
	if websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure) {

		svr.registerCH <- &RegisterPayload{
			client: c,
			cmd:    RegisterCmd_Unregister,
		}
	}

}

type BroadcastWorker struct {
	clients []*Client
	mu      *sync.RWMutex
	msgCh   chan interface{}
	idx     int
}

func NewBroadcastWorker(idx int) *BroadcastWorker {
	return &BroadcastWorker{
		clients: []*Client{},
		mu:      &sync.RWMutex{},
		msgCh:   make(chan interface{}, 64),
		idx:     idx,
	}
}

func (brw *BroadcastWorker) acceptClientLoop(clientCH <-chan *Client) {
	for c := range clientCH {
		brw.mu.Lock()
		brw.clients = append(brw.clients, c)
		brw.mu.Unlock()
		logrus.WithFields(logrus.Fields{
			"wsID":     c.wsID,
			"workerID": brw.idx,
		}).Info("worker added client")
	}
}

func (brw *BroadcastWorker) handleBroadcast() {
	for {
		msg := <-brw.msgCh
		logrus.WithFields(logrus.Fields{
			"workerID": brw.idx,
		}).Info("worker received msg")

		for _, c := range brw.clients {
			err := c.conn.WriteJSON(msg)
			if err != nil {
				// TODO -> provide this func here
				// go svr.initUnregisterOnError(err, c)

				logrus.Errorf("error sending msg to client %s, %v", c.wsID, err)
				continue
			}
			logrus.WithFields(logrus.Fields{
				"wsID": c.wsID,
			}).Info("send msg success")
		}
	}

}

func (svr *WSServer) initBroadcastHub(numWorkers int) {
	for idx := range numWorkers {
		brw := NewBroadcastWorker(idx)
		svr.BroadcastWorkers = append(svr.BroadcastWorkers, brw)
		go brw.acceptClientLoop(svr.clientCH)
		go brw.handleBroadcast()
	}
}

// func (svr *WSServer) BroadcastMSG(msg interface{}) {
// 	logrus.WithFields(logrus.Fields{
// 		"ClientsCount": len(svr.Clients),
// 		"msg":          msg,
// 	}).Infof("broadcasting msg to all clients")

// 	clientsToNotify := []*Client{}
// 	svr.mu.RLock()
// 	for _, c := range svr.Clients {
// 		clientsToNotify = append(clientsToNotify, c)
// 	}
// 	svr.mu.RUnlock()

//		for _, c := range clientsToNotify {
//			err := c.conn.WriteJSON(msg)
//			if err != nil {
//				go svr.initUnregisterOnError(err, c)
//				logrus.Errorf("error sending msg to client %s, %v", c.wsID, err)
//				continue
//			}
//			logrus.WithField("wsID", c.wsID).Info("send msg success")
//		}
//	}

func (svr *WSServer) BroadcastMSG(msg interface{}) {
	logrus.WithFields(logrus.Fields{
		"ClientsCount": len(svr.Clients),
		"msg":          msg,
	}).Infof("broadcasting msg to all clients")

	for _, worker := range svr.BroadcastWorkers {
		worker.msgCh <- msg
	}
}

func (svr *WSServer) SendRoomMSG(data interface{}, clientID string, roomID string) error {
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
	room.mu.RLock()
	for _, v := range room.Clients {
		if v.wsID == clientID {
			validClient = true
		} else {
			clientsToNotify = append(clientsToNotify, v)
		}
	}
	room.mu.RUnlock()
	if !validClient {
		return fmt.Errorf("cliendID: %s does not belong to this roomID: %s", clientID, roomID)
	}
	msg := ClientMSG{
		MsgType: MsgType_RoomMessage,
		Data:    data,
		RoomID:  roomID,
	}

	for _, c := range clientsToNotify {
		err := c.conn.WriteJSON(msg)
		if err != nil {
			go svr.initUnregisterOnError(err, c)
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
	svr.mu.RLock()
	room, ok := svr.Rooms[roomID]
	svr.mu.RUnlock()

	if ok {
		room.mu.Lock()
		client, ok := room.Clients[client.wsID]
		if !ok {
			logrus.Warnf("client %s does not belong to the room %s we are trying to leave", client.wsID, roomID)
		} else {
			delete(room.Clients, client.wsID)
		}
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
	err := client.conn.WriteJSON(msg)
	if err != nil {
		go svr.initUnregisterOnError(err, client)
		logrus.Errorf("err sending ping to the client %v", err)
	}
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
		logrus.WithField("wsID", c.wsID).Info("exiting wsAcceptLoop")
		close(c.msgChan)
	}()

	readCH := make(chan []byte, 64)
	readErrCH := make(chan error, 16)

	go func() {
		defer close(readCH)
		defer close(readErrCH)
		for {
			_, b, err := c.conn.ReadMessage()

			if err != nil {
				logrus.Errorf("error reading conn %v", err)
				readErrCH <- err
				return
			}

			select {
			case readCH <- b:
				logrus.WithField("wsID", c.wsID).Info("readCH -> sending")
			default:
				// TODO -> add rate limit instead of disconnect
				logrus.WithField("wsID", c.wsID).Error("readCH full -> disconnecting")
				err := c.conn.Close()
				if err != nil {
					logrus.Errorf("err closing connection %v", err)
				}
			}

		}
	}()

	for {
		select {
		case <-readErrCH:
			logrus.Warn("exiting wsAcceptLoop, due to err")
			return
		case b := <-readCH:
			c.msgChan <- b
		}
	}
}

func (c *Client) readMsgLoop(svr *WSServer) {
	defer svr.wg.Done()
	svr.wg.Add(1)

	t := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-svr.ctx.Done():
			logrus.Info("received svr close -> starting gracefull shutdown")
			err := c.conn.Close()
			if err != nil {
				logrus.Errorf("err closing connection %v", err)
			}
			return

		case <-t.C:
			count := c.pingCount.Add(-1)
			logrus.Infof("current ping count = %d", count)
			if count < -2 {
				err := c.conn.Close()
				if err != nil {
					logrus.Errorf("err closing connection %v", err)
				}
				return
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

			msg := new(ClientMSG)
			if len(v) == 1 {
				logrus.Infof("received 1 byte msg -> most likely ping")
				msg.MsgType = MsgType_Ping
			} else {
				err := json.Unmarshal(v, msg)
				if err != nil {
					logrus.Errorf("error unmarshaling data %v", err)
					continue
				}
			}

			switch msg.MsgType {
			case MsgType_Broadcast:
				go svr.BroadcastMSG(msg.Data)
			case MsgType_RoomMessage:
				go svr.SendRoomMSG(msg.Data, c.wsID, msg.RoomID)
			case MsgType_JoinRoom:
				svr.JoinRoom(c, msg.RoomID)
			case MsgType_LeaveRoom:
				svr.LeaveRoom(c, msg.RoomID)
			case MsgType_Ping:
				c.pingCount.Add(1)
				svr.Ping(c)
			}
			logrus.WithFields(logrus.Fields{
				"wsID":    c.wsID,
				"msgType": msg.MsgType,
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
//   - 4.add Write Buffering
//   - 5.add rate limiter for clients
//   - 6.make dynamic workers && worker rebalnce on client disconnect
func chatClientLoop() {
	svr := NewWSServer()
	svr.initBroadcastHub(3)

	sigChan := make(chan os.Signal, 1)
	go signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logrus.Info("received exit signal -> init shutdown")
		svr.Cancel()
	}()

	go func() {
		http.HandleFunc("/", svr.handleWS())
		logrus.Info("starting to listen on localhost:2112")
		log.Fatal(http.ListenAndServe("localhost:2112", nil))
	}()

	svr.RegisterLoop()
	logrus.Info("left RegisterLoop -> waiting for all clients to disconnect")
	svr.wg.Wait()
	logrus.Info("EXIT MAIN")
}
