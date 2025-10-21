package main

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Worker struct {
	id      int
	clients map[string]*Client
	mu      *sync.RWMutex
}

func NewWorker(id int) *Worker {
	return &Worker{
		id:      id,
		clients: map[string]*Client{},
		mu:      new(sync.RWMutex),
	}
}

func (w *Worker) SendRoomMsg(msg *Message, cls []*Client, leaveCH chan<- *Client) {
	fmt.Printf("wID = %d, cID = %s\n", w.id, msg.Client.id)
	workerClients := []*Client{}
	w.mu.RLock()
	for _, c := range cls {
		toAdd, ok := w.clients[c.id]
		if ok {
			workerClients = append(workerClients, toAdd)
		}
	}
	w.mu.RUnlock()

	for _, c := range workerClients {
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
					leaveCH <- c
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

func (w *Worker) SendBroadcastMsg(resp *Response, cID string, leaveCH chan<- *Client) {
	fmt.Println("worker ID starting broadcast = ", w.id)
	cls := []*Client{}

	w.mu.RLock()
	for _, c := range w.clients {
		if c.id == cID {
			continue
		}
		cls = append(cls, c)
	}
	w.mu.RUnlock()

	for _, c := range cls {
		fmt.Printf("workerID %d sending msg to clinet = %s\n", w.id, c.id)

		err := c.conn.WriteJSON(resp)
		if err != nil {
			logrus.Error("Error sending msg", err)
			if websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				go func() {
					leaveCH <- c
				}()
			}
		}
	}

	logrus.WithFields(
		logrus.Fields{
			"senderID":      cID,
			"receiverCount": len(cls),
			"workerID":      w.id,
		},
	).Info("sent broadcast")
}
