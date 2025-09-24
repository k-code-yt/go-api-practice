package main

import (
	"encoding/json"
	"fmt"
	"go-logistic-api/shared"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type DataReceiver struct {
	conn      *websocket.Conn
	dataCH    chan []byte
	msgBroker *MsgBroker
}

func NewDataReceiver(conn *websocket.Conn, msgBroker *MsgBroker) *DataReceiver {
	return &DataReceiver{
		conn:      conn,
		dataCH:    make(chan []byte),
		msgBroker: msgBroker,
	}

}

func wsHandler(msgBroker *MsgBroker) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		wsID := r.Header.Get("Sec-Websocket-Key")
		if wsID != "" {
			fmt.Println("Received new WS conn = ", wsID)
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:    1028,
			WriteBufferSize:   1028,
			EnableCompression: true,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
			return
		}

		dr := NewDataReceiver(conn, msgBroker)
		dr.acceptWSLoop()
	}
}

func (dr *DataReceiver) acceptWSLoop() {
	defer dr.conn.Close()
	fmt.Println("----STARTING TO ACCEPT")

	for {
		_, msgBytes, err := dr.conn.ReadMessage()
		if err != nil {
			fmt.Println("error receiving from WS conn ", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				continue
			}
			break
		}

		var sd shared.SensorData
		err = json.Unmarshal(msgBytes, &sd)
		if err != nil {
			fmt.Printf("Error unmarshalling payload %v\n", err)
			continue
		}
		go dr.msgBroker.producer.ProduceData(sd)
	}
}

func main() {
	msgBroker, err := NewMsgBroker()
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", wsHandler(msgBroker))
	logrus.Info("WS:SERVER:STATUS = ", "Ready")
	logrus.Infof("WS:PORT %s\n", shared.WSPort)
	log.Fatal(http.ListenAndServe(shared.WSPort, nil))
}
