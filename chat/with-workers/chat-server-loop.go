package withworkers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"poseur.com/dotenv"
)

var (
	ENV string = ""
)

func loadENV() {
	var envfile = flag.String("env", ".env", "environment file")
	flag.Parse()
	_ = dotenv.SetenvFile(*envfile)
	ENV = os.Getenv("ENV")
}

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
	id     string
	rooms  map[string]*Room
	mu     *sync.RWMutex
	conn   *websocket.Conn
	worker *Worker
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
	exit := make(chan struct{})
	defer func() {
		c.conn.Close()
		srv.leaveServerCH <- c
		logrus.Info("Exiting DEFER FUNC")
	}()

	go func() {
		select {
		case <-exit:
			break
		case <-ctx.Done():
			c.conn.Close()
			break
		}
		logrus.Info("Exiting close GOROUTINE")
	}()

	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			// TODO add test .env to enable/disable
			logrus.Errorf("error reading msg loop for client = %s, err = %v", c.id, err)
			close(exit)
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
	clients          map[string]*Client
	rooms            map[string]*Room
	workers          []*Worker
	clientsPerWorker int

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
	mu       *sync.RWMutex

	// for testing
	activeClients atomic.Int64
}

func NewWSServer(workerCount int) *WSServer {
	ctx, cancel := context.WithCancel(context.Background())
	var workers []*Worker
	for idx := range workerCount {
		w := NewWorker(idx)
		workers = append(workers, w)
	}
	return &WSServer{
		clients:          map[string]*Client{},
		rooms:            map[string]*Room{},
		workers:          workers,
		clientsPerWorker: 50,

		leaveServerCH:      make(chan *Client, 64),
		joinServerCH:       make(chan *Client, 64),
		sendBroadcastMsgCH: make(chan *Message, 64),
		joinRoomCH:         make(chan *Message, 64),
		leaveRoomCH:        make(chan *Message, 64),
		sendRoomMsgCH:      make(chan *Message, 64),
		errCh:              make(chan error, 64),
		shutdownCH:         make(chan struct{}),

		ctx:      ctx,
		cancelFN: cancel,
		wg:       new(sync.WaitGroup),
		mu:       &sync.RWMutex{},
	}
}

// for testing
func (srv *WSServer) GetClientCount() int {
	return int(srv.activeClients.Load())
}

func (srv *WSServer) checkScaleUp() {
	srv.mu.RLock()
	totalClients := len(srv.clients)
	totalWorkers := len(srv.workers)
	srv.mu.RUnlock()

	if totalClients > srv.clientsPerWorker && float64(totalWorkers*srv.clientsPerWorker)*0.5 < float64(totalClients) {
		srv.scaleUp(totalWorkers - 1)
	}
}

func (srv *WSServer) scaleUp(lastIdx int) {
	logrus.Info("ScaleUP got triggered")
	newW := NewWorker(lastIdx + 1)
	srv.mu.Lock()
	srv.workers = append(srv.workers, newW)
	for _, c := range srv.clients {
		w := srv.getWorker(c.id) // new
		if c.worker.id != w.id {
			delete(c.worker.clients, c.id)
			srv.addClientToWorker(c)
		}
	}
	srv.mu.Unlock()
}

func (srv *WSServer) checkScaleDown() {
	srv.mu.RLock()
	totalClients := len(srv.clients)
	totalWorkers := len(srv.workers)
	srv.mu.RUnlock()

	if totalClients > srv.clientsPerWorker && float64(totalWorkers*srv.clientsPerWorker)*0.5 >= float64(totalClients) {
		srv.scaleDown(totalWorkers - 1)
	}
}

func (srv *WSServer) scaleDown(lastIdx int) {
	logrus.Info("ScaleDOWN got triggered")
	srv.mu.Lock()
	srv.workers = srv.workers[:lastIdx]
	for _, c := range srv.clients {
		w := srv.getWorker(c.id) // new
		if c.worker.id != w.id {
			delete(c.worker.clients, c.id)
			c.worker = nil
			srv.addClientToWorker(c)
		}
	}
	srv.mu.Unlock()
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
		srv.joinServerCH <- c
		go c.readMsgLoop(srv.ctx, srv)
	}
}

func (srv *WSServer) getWorker(cID string) *Worker {
	hash := uint32(0)
	for i := 0; i < len(cID); i++ {
		hash = hash*31 + uint32(cID[i])
	}

	workerIndex := int(hash % uint32(len(srv.workers)))
	w := srv.workers[workerIndex]
	return w
}

func (srv *WSServer) AcceptLoop() {
	ENV = os.Getenv("ENV")
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
			// TODO -> add .env switcher
			go srv.SendBroadcastMsgWorkers(msg)
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
		}
	}
}

func (srv *WSServer) addClientToWorker(client *Client) {
	cID := client.id
	w := srv.getWorker(cID)
	w.clients[cID] = client
	client.worker = w
	logrus.WithFields(logrus.Fields{
		"cID":          cID,
		"wID":          w.id,
		"clientsCount": len(w.clients),
	}).Info("client joined worker")
	if mathrand.Intn(50) < 1 {
		fmt.Println("client count on add = ", len(srv.clients))
	}

}

func (srv *WSServer) removeClientFromWorker(client *Client) {
	cID := client.id
	w := srv.getWorker(cID)
	delete(w.clients, cID)
	client.worker = nil
	logrus.WithFields(logrus.Fields{
		"cID":          cID,
		"wID":          w.id,
		"clientsCount": len(w.clients),
	}).Info("client left worker")
	if mathrand.Intn(50) < 1 {
		fmt.Println("client count on remove = ", len(srv.clients))
	}

}

func (srv *WSServer) LeaveServer(client *Client) {
	_, ok := srv.clients[client.id]
	if ok {
		srv.activeClients.Add(-1)
		delete(srv.clients, client.id)
	}

	for _, r := range client.rooms {
		delete(r.clients, client.id)
	}
	client.rooms = nil
	// TODO -> and .env variable to on/off
	srv.removeClientFromWorker(client)
	srv.checkScaleDown()
	// -----
	logrus.WithField("id", client.id).Info("client left server")
}

func (srv *WSServer) JoinServer(client *Client) {
	_, ok := srv.clients[client.id]
	if ok {
		logrus.WithField("id", client.id).Info("cleint already exists")
		return
	}

	// TODO -> and .env variable to on/off
	srv.addClientToWorker(client)
	// -----

	srv.clients[client.id] = client
	srv.activeClients.Add(1)
	srv.checkScaleUp()
	// logrus.WithField("id", client.id).Info("client joined server")
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
	w := srv.getWorker(msg.Client.id)
	fmt.Printf("wID = %d, cID = %s\n", w.id, msg.Client.id)
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
				srv.leaveServerCH <- msg.Client
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

func (srv *WSServer) SendRoomMsgWorkers(msg *Message) {
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

	for _, w := range srv.workers {
		go w.SendRoomMsg(msg, cls, srv.leaveServerCH)
	}
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
				srv.leaveServerCH <- msg.Client
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

func (srv *WSServer) SendBroadcastMsgWorkers(msg *Message) {
	srv.mu.RLock()
	msg.RoomID = ""
	cID := msg.Client.id
	for _, w := range srv.workers {
		resp := NewResponse(msg)
		go w.SendBroadcastMsg(resp, cID, srv.leaveServerCH)
	}
	srv.mu.RUnlock()
}

func (srv *WSServer) cleanUp() {
	close(srv.leaveServerCH)
	close(srv.joinServerCH)
	close(srv.sendBroadcastMsgCH)
	close(srv.joinRoomCH)
	close(srv.leaveRoomCH)
	close(srv.sendRoomMsgCH)
	close(srv.errCh)
}

func chatServer() {
	wsSrv := NewWSServer(1)
	defer wsSrv.cancelFN()
	http.HandleFunc("/", prometheusMiddleware(wsSrv.wsHandler()))
	go wsSrv.AcceptLoop()
	port := ":2112"
	logrus.Infof("HTTP server is listening on %s\n", port)
	logrus.Fatal(http.ListenAndServe(port, nil))
}
