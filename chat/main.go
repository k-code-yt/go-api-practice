package main

import (
	"crypto/rand"
	"log"
	"net/http"
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
	MsgType_JoinRoom  MsgType = "join-room"
	MsgType_LeaveRoom MsgType = "leave-room"
	MsgType_Broadcast MsgType = "broadcast"
	MsgType_Direct    MsgType = "direct"
)

type RegisterCmd string

const (
	RegisterCmd_Register   RegisterCmd = "register"
	RegisterCmd_Unregister RegisterCmd = "unregister"
)

type Room struct {
	Clients []*Client
}

type Client struct {
	conn    *websocket.Conn
	msgChan chan *ClientMSG
	wsID    string
}

func NewClient(conn *websocket.Conn) *Client {
	wsID := rand.Text()
	logrus.Infof("new client conn %s", wsID)
	return &Client{
		conn:    conn,
		msgChan: make(chan *ClientMSG),
		wsID:    wsID,
	}
}

type RegisterPayload struct {
	client *Client
	cmd    RegisterCmd
}

type BroadcastServer struct {
	Clients    []*Client
	mu         *sync.Mutex
	registerCH chan *RegisterPayload
}

func NewBroadcastServer() *BroadcastServer {
	var clientSlice []*Client
	return &BroadcastServer{
		Clients:    clientSlice,
		mu:         new(sync.Mutex),
		registerCH: make(chan *RegisterPayload, 24),
	}
}

func (brs *BroadcastServer) Register(client *Client) {
	brs.mu.Lock()
	defer brs.mu.Unlock()
	logrus.Infof("client joined broadcast %s", client.wsID)
	brs.Clients = append(brs.Clients, client)

}

func (brs *BroadcastServer) Unregister(client *Client) {
	brs.mu.Lock()
	defer brs.mu.Unlock()
	var clientsCopy []*Client
	for _, v := range brs.Clients {
		if v.wsID == client.wsID {
			continue
		}
		clientsCopy = append(clientsCopy, v)
	}

	brs.Clients = nil
	brs.Clients = clientsCopy
	logrus.Infof("client removed from broadcast %s", client.wsID)
}

func (brs *BroadcastServer) RegisterLoop() {
	defer close(brs.registerCH)
	for {
		v := <-brs.registerCH
		if v.cmd == RegisterCmd_Register {
			brs.Register(v.client)
		} else {
			brs.Unregister(v.client)
		}
	}
}

type ClientMSG struct {
	MsgType MsgType     `json:"type"`
	Data    interface{} `json:"data"`
}

func handleWS() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
		brs := NewBroadcastServer()
		go brs.RegisterLoop()

		client := NewClient(conn)

		go func() {
			brs.registerCH <- &RegisterPayload{
				client: client,
				cmd:    RegisterCmd_Register,
			}
		}()

		go client.readMsgLoop()
		time.Sleep(time.Millisecond * 50)
		go client.wsAcceptLoop(brs.registerCH)
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

func (c *Client) readMsgLoop() {
	for v := range c.msgChan {
		// TODO -> add join room logic, based on room
		logrus.Info(v)
	}
}

func main() {
	http.HandleFunc("/", handleWS())
	logrus.Info("starting to listen on localhost:3231")
	log.Fatal(http.ListenAndServe("localhost:3231", nil))
}
